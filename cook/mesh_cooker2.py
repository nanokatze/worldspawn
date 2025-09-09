# Here we use our own terminology, rather than Blender's

import dataclasses
import numpy as np
import json
import io
import utils
import numpy_utils as nputils
import struct


class Raw:


    def __init__(self, materials, tris, tri_mat_idxs):
        # TODO: should these be private?
        self.materials = materials
        self.tris = tris
        self.tri_mat_idxs = tri_mat_idxs


@dataclasses.dataclass
class Collision:
    VertexBuffer: int
    VertexCount: int
    TriangleBuffer: int
    TriangleCount: int


@dataclasses.dataclass
class Part:
    MaterialIndex: int
    PosBuffer: int
    NormalBuffer: int
    AttribBuffers: list[int]
    VertexCount: int
    IndexType: str
    IndexBuffer: int
    TriangleCount: int


@dataclasses.dataclass
class Rendering:
    Parts: list[Part]


@dataclasses.dataclass
class Header:
    Materials: list[str]
    Collision: Collision
    Rendering: Rendering


def cook(raw, directory):
    collision = None
    rendering = None
    blob = io.BytesIO() # TODO: use a stricter alignment when writing to blob

    if True:
        verts_unindexed = raw.tris.reshape(-1)['position']
        verts_indexed, vert_idxs = np.unique(verts_unindexed, return_inverse=True, axis=0)

        vertex_buffer = seek_align(blob, 4)
        nputils.write_ndarray(blob, verts_indexed)

        tri_buffer = seek_align(blob, 4)
        # TODO: probably deinterleave?
        nputils.write_ndarray(blob, np.c_[
            vert_idxs.astype('<u4').reshape((-1, 3)),
            raw.tri_mat_idxs,
        ])

        collision = Collision(
            VertexBuffer=vertex_buffer,
            VertexCount=len(verts_indexed),
            TriangleBuffer=tri_buffer,
            TriangleCount=len(vert_idxs)//3,
        )

    if True:
        parts = []
        for material_index in np.unique(raw.tri_mat_idxs):
            assert material_index < len(raw.materials)

            tris = raw.tris[raw.tri_mat_idxs == material_index]
            if len(tris) == 0:
                continue

            verts_unindexed = tris.flat

            verts_indexed, vert_idxs = np.unique(verts_unindexed, return_inverse=True)

            index_size = 2 if len(verts_indexed) <= 65535 else 4

            # TODO: check whether indices are actually a benefit

            index_buffer = seek_align(blob, 4)
            nputils.write_ndarray(blob, vert_idxs.astype(f'<u{index_size}'))

            verts = verts_indexed

            pos_buffer = seek_align(blob, 4)
            nputils.write_ndarray(blob, verts['position'])

            normal_buffer = seek_align(blob, 4)
            nputils.write_ndarray(blob, verts['normal'])

            attrib_buffers = []

            attrib_buffers.append(seek_align(blob, 4))
            nputils.write_ndarray(blob, verts['UVMap'])

            parts.append(Part(
                MaterialIndex=int(material_index),
                PosBuffer=pos_buffer,
                NormalBuffer=normal_buffer,
                AttribBuffers=attrib_buffers,
                VertexCount=len(verts),
                IndexType={2: 'UINT16', 4: 'UINT32'}[index_size], # TODO: factor this map out pls
                IndexBuffer=index_buffer,
                TriangleCount=len(tris),
            ))

        rendering = Rendering(Parts=parts)

    with open(directory, 'wb') as f:
        f.write(b'Worldspawn')
        seek_align(f, 16)
        f.write(b'Geometry')
        seek_align(f, 16)

        sections = f.tell()
        f.write(struct.pack('<qqqq', 0, 0, 0, 0))

        json_offset = f.seek(0, 1)
        h = Header(
            Materials=raw.materials,
            Collision=collision,
            Rendering=rendering,
        )
        d = dataclasses.asdict(h, dict_factory=dict_skip_nulls)
        d = fixupdict(d)
        json.dump(d, utils.UTF8Writer(f), default=utils.asdasd)
        json_length = f.seek(0, 1) - json_offset

        blob_offset = f.seek(0, 1)
        f.write(blob.getbuffer())
        blob_length = f.seek(0, 1) - blob_offset

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
# TODO: this should be like seek (i.e. take offset and whence) but also take align
# TODO: rename to something else? this also does write, which isn't really
# appropriate for something called seek? :thinking:
def seek_align(f, align=1):
    off = f.seek(0, 1)
    pad = -off % align
    if pad > 0:
        f.write(b'\x00' * pad)
        off += pad
    return off
