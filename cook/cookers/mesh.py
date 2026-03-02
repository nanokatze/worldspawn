import dataclasses
import numpy as np
import json
import io
import enum
import struct

import numpyutil as nputil
import util
import bpyutil


# TODO: focus on just exporting anything at all here and make optimization etc
# be the responsibility of a Go program?


# TODO: should this be a plain int?
class Domain(enum.Enum):
    VERTEX = 0
    EDGE = 1
    FACE = 2


@dataclasses.dataclass
class AttributeBuffer:
    domain: Domain
    data: object


@dataclasses.dataclass
class Raw:
    positions: AttributeBuffer
    normals: AttributeBuffer

    joints: list[str]

    joint_weights: object

    materials: list[str]

    material_indices: AttributeBuffer

    named_attributes: dict[str, AttributeBuffer]


# TODO: we can prefix the internal stuff that goes into the file with "json"


@dataclasses.dataclass
class _Buffer:
    Data: int
    Size: int


@dataclasses.dataclass
class _AttributeBuffer:
    Type: str # TODO: make this a VkFormat?
    Domain: int # TODO: make this a string
    Data: int
    # Size: int


@dataclasses.dataclass
class _Range:
    MaterialIndex: int # TODO: kill
    First: int
    Count: int


@dataclasses.dataclass
class _Header:
    PrimitiveCount: int

    VertexCount: int

    IndexType: int # TODO: make this a string? aaaaaaaa

    IndexBuffer: int

    AttributeBuffers: list[_AttributeBuffer]

    PositionAttribute: int

    NormalAttribute: int

    Joints: list[str]

    JointWeights: _Buffer

    Materials: list[str]

    # MaterialIndexAttribute: int

    MaterialIndexRanges: list[_Range]

    NamedAttributes: dict[str, int]


def cook(raw, directory):
    # Validation
    #
    # TODO: more elaborate validation

    all_attributes = [
        raw.positions,
        raw.normals,
        raw.material_indices,
    ]

    primitive_count = len(all_attributes[0].data)

    for i, buf in enumerate(all_attributes):
        assert len(buf.data) == primitive_count

    blob = io.BytesIO() # TODO: use a stricter alignment when writing to blob

    # Sort triangles by material index

    material_index_sorter = np.argsort(raw.material_indices.data, kind='stable')

    for k, v in enumerate(all_attributes):
        all_attributes[k].data = v.data[material_index_sorter]

    index_type = 2

    index_buffer = seek_align(blob, 4)
    nputil.tofile(blob, np.arange(3 * primitive_count).astype(f'<u{index_type}'))

    attribute_buffers = []
    attribute_remap = {}
    for i, buf in enumerate(all_attributes):
        types = {
            np.dtype(('<f4', (3,))): 'R32G32B32_SFLOAT',
            np.dtype(('<f4', (2,))): 'R32G32_SFLOAT',
            # np.dtype(('<u4', 1,)): 'R32_UINT',
        }

        if buf.domain == Domain.VERTEX:
            attribute_buffers.append(_AttributeBuffer(
                Domain=0,
                Type=types[np.dtype((buf.data.dtype, buf.data.shape[2:]))],
                Data=seek_align(blob, 4))) # TODO: do seek_align before pls
            attribute_remap[i] = len(attribute_buffers) - 1

            nputil.tofile(blob, buf.data)

    # named_attributes = {name: attribute_remap[index] for (name, index) in raw.named_attributes.items() if index in attribute_remap}

    # TODO: assert that primitives are sorted by material index
    material_index_ranges = []
    # TODO: remove the MaterialIndex indirection
    for material_indices in np.unique(raw.material_indices.data):
        a = np.searchsorted(raw.material_indices.data, material_indices)
        b = np.searchsorted(raw.material_indices.data, material_indices, side='right')
        material_index_ranges.append(_Range(int(material_indices), int(a), int(b - a)))

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
            VertexCount=primitive_count * 3,
            IndexType=index_type,
            IndexBuffer=index_buffer,
            AttributeBuffers=attribute_buffers,
            PositionAttribute=0, # TODO: avoid hardcoding these
            NormalAttribute=1,
            Joints=raw.joints,
            JointWeights=None,
            Materials=raw.materials,
            MaterialIndexRanges=material_index_ranges,
            NamedAttributes={},
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
