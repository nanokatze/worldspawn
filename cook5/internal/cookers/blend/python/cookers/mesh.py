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
# TODO: alternatively get rid of all the dataclass stuff? We should see what's
# the "idiomatic" way to write jsons in python is.


@dataclasses.dataclass
class _Buffer:
    Data: int
    Size: int


@dataclasses.dataclass
class _AttributeBuffer:
    Domain: int # TODO: make this a string
    Type: str # TODO: make this a VkFormat?
    Data: _Buffer


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

    IndexBuffer: _Buffer

    Positions: _AttributeBuffer
    Normals: _AttributeBuffer

    Joints: list[str]

    MaxInfluencesPerVertex: int

    JointWeights: _Buffer

    Materials: list[str]

    # MaterialIndices: _AttributeBuffer

    MaterialIndexRanges: list[_Range]

    NamedAttributes: dict[str, _AttributeBuffer]


def _write_buffer(blob, buf):
    off = seek_align(blob, 4)
    size = nputil.tofile(blob, buf)
    return _Buffer(Data=off, Size=size)


_types = {
    np.dtype(('<f4', (3,))): 'R32G32B32_SFLOAT',
    np.dtype(('<f4', (2,))): 'R32G32_SFLOAT',
    # np.dtype(('<u4', 1,)): 'R32_UINT',
}


def _write_attribute_buffer(blob, buf):
    assert buf.domain == Domain.VERTEX

    return _AttributeBuffer(
        Domain=0,
        # TODO: for this to work we actually need to agree upon the shapes
        # between different domains, or have the user specify the format.
        Type=_types[np.dtype((buf.data.dtype, buf.data.shape[2:]))],
        Data=_write_buffer(blob, buf.data))


def _sort_by_material_index(raw):
    attribute_buffers = [
        raw.positions,
        raw.normals,
        raw.joint_weights,
        raw.material_indices,
        *raw.named_attributes.values(),
    ]

    material_index_sorter = np.argsort(raw.material_indices.data, kind='stable')

    for k, v in enumerate(attribute_buffers):
        # TODO: rewrite this to be less ass
        attribute_buffers[k].data = v.data[material_index_sorter]


def cook(ctx, raw, directory):
    attribute_buffers = [
        raw.positions,
        raw.normals,
        raw.joint_weights,
        raw.material_indices,
        *raw.named_attributes.values(),
    ]

    primitive_count = len(attribute_buffers[0].data)

    # Validation

    for i, buf in enumerate(attribute_buffers):
        assert len(buf.data) == primitive_count

        match buf.domain:
            case Domain.VERTEX:
                assert buf.data.shape[1] == 3
            case Domain.FACE:
                pass
            case _:
                assert False, 'unreachable'

    # Sort by material index and write out the ranges

    _sort_by_material_index(raw)

    material_index_ranges = []
    # TODO: remove the MaterialIndex field
    for material_indices in np.unique(raw.material_indices.data):
        a = np.searchsorted(raw.material_indices.data, material_indices)
        b = np.searchsorted(raw.material_indices.data, material_indices, side='right')
        material_index_ranges.append(_Range(int(material_indices), int(a), int(b - a)))

    # Reindex things (TODO: actually reindex things)

    indices = np.arange(3 * primitive_count)

    index_type = 4

    # Joint weights
    # TODO: compact joint weights

    max_influences_per_vertex = raw.joint_weights.data.shape[2]

    # Write stuff out

    blob = io.BytesIO()

    index_buffer = _write_buffer(blob, indices.astype(f'<u{index_type}'))

    positions = _write_attribute_buffer(blob, raw.positions)
    normals = _write_attribute_buffer(blob, raw.normals)

    # Basically a weird attribute
    joint_weights = _write_buffer(blob, raw.joint_weights.data)

    named_attributes = {}
    for name in sorted(raw.named_attributes.keys()):
        attr_buf = raw.named_attributes[name]
        # TODO: make things function for Domain.FACE as well
        if attr_buf.domain == Domain.VERTEX:
            continue
        named_attributes[name] = _write_attribute_buffer(blob, attr_buf)

    with ctx.create(directory) as f:
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
            Positions=positions,
            Normals=normals,
            Joints=raw.joints,
            MaxInfluencesPerVertex=max_influences_per_vertex,
            JointWeights=joint_weights,
            Materials=raw.materials,
            MaterialIndexRanges=material_index_ranges,
            NamedAttributes=named_attributes,
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
