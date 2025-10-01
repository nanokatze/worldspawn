# Here we use our own terminology, rather than Blender's

import dataclasses
import numpy as np
import json
import io
import struct

import numpyutil as nputil
import util
import bpyutil


@dataclasses.dataclass
class Raw:
    _materials: list[str]
    _tris: object
    _tri_mat_idxs: object


@dataclasses.dataclass
class _Collision:
    VertexBuffer: int
    VertexCount: int
    TriangleBuffer: int
    TriangleCount: int


@dataclasses.dataclass
class _Part:
    MaterialIndex: int
    PosBuffer: int
    NormalBuffer: int
    AttribBuffers: list[int]
    VertexCount: int
    IndexType: str
    IndexBuffer: int
    TriangleCount: int


@dataclasses.dataclass
class _Rendering:
    Parts: list[_Part]


@dataclasses.dataclass
class _Header:
    Materials: list[str]
    Collision: _Collision
    Rendering: _Rendering


def cook(raw, directory):
    collision = None
    rendering = None
    blob = io.BytesIO() # TODO: use a stricter alignment when writing to blob

    if True:
        verts_unindexed = raw._tris.reshape(-1)['position']
        verts_indexed, vert_idxs = np.unique(verts_unindexed, return_inverse=True, axis=0)

        vertex_buffer = seek_align(blob, 4)
        nputil.tofile(blob, verts_indexed)

        tri_buffer = seek_align(blob, 4)
        # TODO: probably deinterleave?
        nputil.tofile(blob, np.c_[
            vert_idxs.astype('<u4').reshape((-1, 3)),
            raw._tri_mat_idxs,
        ])

        collision = _Collision(
            VertexBuffer=vertex_buffer,
            VertexCount=len(verts_indexed),
            TriangleBuffer=tri_buffer,
            TriangleCount=len(vert_idxs)//3,
        )

    if True:
        parts = []
        for material_index in np.unique(raw._tri_mat_idxs):
            assert material_index < len(raw._materials)

            tris = raw._tris[raw._tri_mat_idxs == material_index]
            if len(tris) == 0:
                continue

            verts_unindexed = tris.flat

            verts_indexed, vert_idxs = np.unique(verts_unindexed, return_inverse=True)

            index_size = 2 if len(verts_indexed) <= 65535 else 4

            # TODO: check whether indices are actually a benefit

            index_buffer = seek_align(blob, 4)
            nputil.tofile(blob, vert_idxs.astype(f'<u{index_size}'))

            verts = verts_indexed

            pos_buffer = seek_align(blob, 4)
            nputil.tofile(blob, verts['position'])

            normal_buffer = seek_align(blob, 4)
            nputil.tofile(blob, verts['normal'])

            attrib_buffers = []

            attrib_buffers.append(seek_align(blob, 4))
            nputil.tofile(blob, verts['UVMap'])

            parts.append(_Part(
                MaterialIndex=int(material_index),
                PosBuffer=pos_buffer,
                NormalBuffer=normal_buffer,
                AttribBuffers=attrib_buffers,
                VertexCount=len(verts),
                IndexType={2: 'UINT16', 4: 'UINT32'}[index_size], # TODO: factor this map out pls
                IndexBuffer=index_buffer,
                TriangleCount=len(tris),
            ))

        rendering = _Rendering(Parts=parts)

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
