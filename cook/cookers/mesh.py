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
    _attrs: object # TODO: write it out properly: _attributes


# TODO: we can prefix the internal stuff that goes into the file, with "json"


# TODO: rename
@dataclasses.dataclass
class _AttributeDesc:
    Name: str
    Type: str
    Domain: int # TODO: should this be an int degree?
    Data: int


@dataclasses.dataclass
class _Part:
    MaterialIndex: int
    FirstPrimitive: int
    PrimitiveCount: int


@dataclasses.dataclass
class _Header:
    PrimitiveCount: int

    VertexCount: int

    IndexType: str
    IndexBuffer: int

    Attributes: list[_AttributeDesc]

    Materials: list[str]

    # Ad-hoc structures

    # TODO: rename 😭
    PartitionByMaterialIndex: list[_Part]


def cook(raw, directory):
    # TODO: perform validation

    attr0 = raw._attrs['position']
    for name, desc in raw._attrs.items():
        assert len(desc._data) == len(attr0._data)

    blob = io.BytesIO() # TODO: use a stricter alignment when writing to blob

    # Sort triangles by material_index
    # TODO: use stable=True when we can use numpy>=2.0.0
    sorted_tris = np.argsort(raw._attrs['material_index']._data)

    for k, v in raw._attrs.items():
        raw._attrs[k]._data = v._data[sorted_tris]

    tri_count = len(raw._attrs['position']._data)

    index_size = 2

    index_buffer = seek_align(blob, 4)
    nputil.tofile(blob, np.arange(3 * tri_count).astype(f'<u{index_size}'))

    attributes = [] # TODO: write it out as a map
    for name, desc in sorted(raw._attrs.items(), key=lambda it: it[0]):
        types = {
            np.dtype(('<f4', (3,))): 'R32G32B32_SFLOAT',
            np.dtype(('<f4', (2,))): 'R32G32_SFLOAT',
            # np.dtype(('<u4', 1,)): 'R32_UINT',
        }

        if desc._domain == Domain.VERTEX:
            attributes.append(_AttributeDesc(
                Name=name,
                Type=types[np.dtype((desc._data.dtype, desc._data.shape[2:]))],
                Domain=0,
                Data=seek_align(blob, 4))) # TODO: do seek_align before pls

            nputil.tofile(blob, raw._attrs[name]._data)

    # TODO: assert that material_index is sorted
    parts = []
    # TODO: remove Part2.MaterialIndex indirection
    for material_index in np.unique(raw._attrs['material_index']._data):
        a = int(np.searchsorted(raw._attrs['material_index']._data, material_index))
        b = int(np.searchsorted(raw._attrs['material_index']._data, material_index, side='right'))
        # assert len(parts) == material_index
        parts.append(_Part(int(material_index), a, b - a))

    with open(directory, 'wb') as f:
        f.write(b'Worldspawn')
        seek_align(f, 16)
        f.write(b'Geometry')
        seek_align(f, 16)

        sections = f.tell()
        f.write(struct.pack('<qqqq', 0, 0, 0, 0))

        json_offset = f.tell()
        h = _Header(
            PrimitiveCount=tri_count,

            VertexCount=3 * tri_count,

            IndexType={2: 'UINT16', 4: 'UINT32'}[index_size], # TODO: factor this map out pls
            IndexBuffer=index_buffer,

            Attributes=attributes,

            Materials=raw._materials,

            PartitionByMaterialIndex=parts,
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
