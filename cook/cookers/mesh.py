import dataclasses
import numpy as np
import json
import io
import enum
import struct

import numpyutil as nputil
import util
import bpyutil


@dataclasses.dataclass
class FaceAttributes:
    _data: object


@dataclasses.dataclass
class VertexAttributes:
    _data: object


@dataclasses.dataclass
class Raw:
    _materials: list[str]
    _attrs: object


# TODO: we can prefix the internal stuff that goes into the file, with "json"


@dataclasses.dataclass
class _Collision:
    VertexBuffer: int
    VertexCount: int
    TriangleBuffer: int
    TriangleCount: int


@dataclasses.dataclass
class _Part:
    MaterialIndex: int
    AttribBuffers: list[int]
    VertexCount: int
    # TODO: factor it out to the top level, next to attribute descs
    IndexType: str
    IndexBuffer: int
    TriangleCount: int


@dataclasses.dataclass
class _AttributeDesc:
    Domain: str
    Name: str
    Type: str


@dataclasses.dataclass
class _Rendering:
    Attributes: list[_AttributeDesc]
    Parts: list[_Part]


@dataclasses.dataclass
class _Header:
    Materials: list[str]
    Collision: _Collision
    Rendering: _Rendering


def cook(raw, directory):
    # TODO: perform validation

    attr0 = raw._attrs['position']
    for name, desc in raw._attrs.items():
        assert len(desc._data) == len(attr0._data)

    collision = None
    rendering = None
    blob = io.BytesIO() # TODO: use a stricter alignment when writing to blob

    if True:
        verts_unindexed = raw._attrs['position']._data.reshape((-1, 3))
        verts_indexed, vert_idxs = np.unique(verts_unindexed, return_inverse=True, axis=0)

        vertex_buffer = seek_align(blob, 4)
        nputil.tofile(blob, verts_indexed)

        tri_buffer = seek_align(blob, 4)
        # TODO: probably deinterleave?
        nputil.tofile(blob, np.c_[
            vert_idxs.astype('<u4').reshape((-1, 3)),
            raw._attrs['material_index']._data,
        ])

        collision = _Collision(
            VertexBuffer=vertex_buffer,
            VertexCount=len(verts_indexed),
            TriangleBuffer=tri_buffer,
            TriangleCount=len(vert_idxs)//3,
        )

    if True:
        attributes = []
        for name, desc in sorted(raw._attrs.items(), key=lambda it: it[0]):
            types = {
                np.dtype(('<f4', (3,))): 'R32G32B32_SFLOAT',
                np.dtype(('<f4', (2,))): 'R32G32_SFLOAT',
                # np.dtype(('<u4', 1,)): 'R32_UINT',
            }
            if isinstance(desc, VertexAttributes):
                attributes.append(_AttributeDesc(Name=name, Type=types[np.dtype((desc._data.dtype, desc._data.shape[2:]))], Domain='VERTEX'))

        parts = []
        for material_index in np.unique(raw._attrs['material_index']._data):
            assert material_index < len(raw._materials)

            tri_idxs = raw._attrs['material_index']._data == material_index
            tri_count = int(tri_idxs.sum())
            if tri_count == 0:
                continue

            assert tri_count > 0

            index_size = 2

            index_buffer = seek_align(blob, 4)
            nputil.tofile(blob, np.arange(3 * tri_count).astype(f'<u{index_size}'))

            attrib_buffers = []
            for attr in attributes:
                attrib_buffers.append(seek_align(blob, 4))
                nputil.tofile(blob, raw._attrs[attr.Name]._data[tri_idxs])

            parts.append(_Part(
                MaterialIndex=int(material_index),
                AttribBuffers=attrib_buffers,
                VertexCount=3 * tri_count,
                IndexType={2: 'UINT16', 4: 'UINT32'}[index_size], # TODO: factor this map out pls
                IndexBuffer=index_buffer,
                TriangleCount=tri_count,
            ))

        rendering = _Rendering(Attributes=attributes, Parts=parts)

    with open(directory, 'wb') as f:
        f.write(b'Worldspawn')
        seek_align(f, 16)
        f.write(b'Geometry')
        seek_align(f, 16)

        sections = f.tell()
        f.write(struct.pack('<qqqq', 0, 0, 0, 0))

        json_offset = f.tell()
        h = _Header(
            Materials=raw._materials,
            Collision=collision,
            Rendering=rendering,
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
