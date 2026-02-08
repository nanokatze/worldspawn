import dataclasses
import numpy as np
import json
import io
import enum
import struct

import numpyutil as nputil
import util
import bpyutil



# TODO: should this be a plain int?
class Domain(enum.Enum):
    VERTEX = 0
    EDGE = 1
    FACE = 2


@dataclasses.dataclass
class AttributeBuffer:
    _domain: int
    _data: object


@dataclasses.dataclass
class Raw:
    _materials: list[str]
    _attributes: dict[str, AttributeBuffer]


# TODO: we can prefix the internal stuff that goes into the file, with "json"


@dataclasses.dataclass
class _Attribute:
    Name: str
    Type: str
    Domain: int # TODO: should this be an int degree?
    Data: int


@dataclasses.dataclass
class _Range:
    MaterialIndex: int # TODO: kill
    First: int
    Count: int


@dataclasses.dataclass
class _Header:
    PrimitiveCount: int

    VertexCount: int

    IndexType: str
    IndexBuffer: int

    Attributes: list[_Attribute]

    Materials: list[str]

    # Ad-hoc structures

    RangesByMaterialIndex: list[_Range]


def cook(raw, directory):
    # Validation
    #
    # TODO: more elaborate validation

    primitive_count = len(raw._attributes['position']._data)

    for name, buf in raw._attributes.items():
        assert len(buf._data) == primitive_count

    blob = io.BytesIO() # TODO: use a stricter alignment when writing to blob

    # Sort triangles by material_index

    material_index_sorter = np.argsort(raw._attributes['material_index']._data, kind='stable')

    for k, v in raw._attributes.items():
        raw._attributes[k]._data = v._data[material_index_sorter]

    index_size = 2

    index_buffer = seek_align(blob, 4)
    nputil.tofile(blob, np.arange(3 * primitive_count).astype(f'<u{index_size}'))

    # Iterate over attributes in a particular order so that we write stuff out consistently.

    attributes = []
    for name, buf in sorted(raw._attributes.items(), key=lambda it: it[0]):
        types = {
            np.dtype(('<f4', (3,))): 'R32G32B32_SFLOAT',
            np.dtype(('<f4', (2,))): 'R32G32_SFLOAT',
            # np.dtype(('<u4', 1,)): 'R32_UINT',
        }

        if buf._domain == Domain.VERTEX:
            attributes.append(_Attribute(
                Name=name,
                Type=types[np.dtype((buf._data.dtype, buf._data.shape[2:]))],
                Domain=0,
                Data=seek_align(blob, 4))) # TODO: do seek_align before pls

            nputil.tofile(blob, buf._data)

    # TODO: assert that primitives are sorted by material_index
    ranges_by_material_index = []
    # TODO: remove the MaterialIndex indirection
    for material_index in np.unique(raw._attributes['material_index']._data):
        a = np.searchsorted(raw._attributes['material_index']._data, material_index)
        b = np.searchsorted(raw._attributes['material_index']._data, material_index, side='right')
        ranges_by_material_index.append(_Range(int(material_index), int(a), int(b - a)))

    with open(directory, 'wb') as f:
        f.write(b'Worldspawn')
        seek_align(f, 16)
        f.write(b'Mesh')
        seek_align(f, 16)

        sections = f.tell()
        f.write(struct.pack('<qqqq', 0, 0, 0, 0))

        json_offset = f.tell()
        h = _Header(
            PrimitiveCount=primitive_count,

            VertexCount=3 * primitive_count,

            IndexType={2: 'UINT16', 4: 'UINT32'}[index_size], # TODO: factor this map out pls
            IndexBuffer=index_buffer,

            Attributes=attributes,

            Materials=raw._materials,

            RangesByMaterialIndex=ranges_by_material_index,
        )
        d = dataclasses.asdict(h, dict_factory=dict_skip_nulls)
        d = fixupdict(d)
        json.dump(d, util.UTF8Writer(f), default=bpyutil.asdasd)
        json_length = f.tell() - json_offset

        blob_offset = f.tell()
        f.write(blob.getbuffer())
        blob_length = f.tell() - blob_offset

        f.seek(sections)
        f.write(struct.pack('<qqqq', json_offset, json_length, blob_offset, blob_length))

# TODO: move to utils somewhere
def dict_skip_nulls(stuff):
    return dict((k, v) for (k, v) in stuff if v is not None)

# TODO: rename to dict_stringify_numbers or something idk
# TODO: alternatively just get rid of this and think of something else?
def fixupdict(d):
    if isinstance(d, dict):
        for k, v in d.items():
            d[k] = fixupdict(v)
    elif isinstance(d, (list, tuple)):
        for k, v in enumerate(d):
            d[k] = fixupdict(v)
    elif isinstance(d, (int, float)):
        d = str(d)
    return d


# TODO: move into utils or something like that package
def seek_align(f, align=1):
    return f.seek(-f.tell() % align, 1)
