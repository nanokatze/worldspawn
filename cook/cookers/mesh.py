import dataclasses
import numpy as np
import json
import io
import struct

import numpyutil as nputil
import util
import bpyutil


# TODO: switch wmesh and other things to use nice (i.e. basically struct.pack)


@dataclasses.dataclass
class Raw:
    # TODO: also pass attribute desc map
    _materials: list[str]
    _tris: object
    _tri_mat_idxs: object # TODO: kill this and fold into tris instead


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
    Name: str
    Type: str
    # Domain:


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
        attributes = []
        for name, (dtype, size) in raw._tris.dtype.fields.items():
            # TODO: gross, we should just take the attribute desc map from the
            # user
            types = {
                np.dtype(('<f4', (3,))): 'R32G32B32_SFLOAT',
                np.dtype(('<f4', (2,))): 'R32G32_SFLOAT',
            }
            attributes.append(_AttributeDesc(Name=name, Type=types[dtype]))

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

            attrib_buffers = []
            for name in raw._tris.dtype.fields:
                attrib_buffers.append(seek_align(blob, 4))
                nputil.tofile(blob, verts[name])

            parts.append(_Part(
                MaterialIndex=int(material_index),
                AttribBuffers=attrib_buffers,
                VertexCount=len(verts),
                IndexType={2: 'UINT16', 4: 'UINT32'}[index_size], # TODO: factor this map out pls
                IndexBuffer=index_buffer,
                TriangleCount=len(tris),
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
