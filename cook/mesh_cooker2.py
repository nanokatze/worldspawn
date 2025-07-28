# Here we use our own terminology, rather than Blender's

import dataclasses
import numpy as np
import json
import util
import struct


@dataclasses.dataclass
class Collider:
    Material: int
    Vertices: int
    VertexCount: int
    Triangles: int
    TriangleCount: int


@dataclasses.dataclass
class Part:
    Positions: int
    Normals: int
    Attributes: list[int]
    VertexCount: int
    IndexType: str
    Triangles: int
    TriangleCount: int



@dataclasses.dataclass
class Header:
    Materials: list[int]
    Collider: Collider
    Rendering: list[Part]



class Raw:


    def __init__(self, tris, tri_mat_idxs):
        self.tris = tris
        self.tri_mat_idxs = tri_mat_idxs


# TODO: move out from this file
class ByteBuffer:


    def __init__(self):
        self.buf = bytearray()


    def seek(self, offset, whence=1):
        assert offset == 0
        assert whence == 1
        return len(self.buf)


    def write(self, b):
        self.buf.extend(b)


    def bytes(self):
        return self.buf


class UTF8Writer:


    def __init__(self, w):
        self.__w = w


    def write(self, s):
        return self.__w.write(s.encode('utf-8'))


def cook(raw, directory):
    collider = None
    parts = []
    blob = ByteBuffer() # TODO: use a stricter alignment when writing to blob

    if True:
        verts_unindexed = np.asarray(raw.tris.flat)['position']
        verts_indexed, vert_idxs = np.unique(verts_unindexed, return_inverse=True, axis=0)

        vertex_buffer = seek_align(blob, 4)
        write_ndarray(blob, verts_indexed)

        tri_buffer = seek_align(blob, 4)
        write_ndarray(blob, np.c_[
            vert_idxs.reshape((-1, 3)),
            raw.tri_mat_idxs,
        ])

        collider = Collider(0, vertex_buffer, len(verts_indexed), tri_buffer, len(vert_idxs)//3)

    #for material_index in range(np.max(raw.tri_mat_idxs)):
    #    tris = raw.tris[raw.tri_mat_idxs == material_index]

    if True:
        tris = raw.tris
        assert len(tris) > 0 # TODO: this CAN happen, handle the case when there's no tris to write

        verts_unindexed = tris.flat

        verts_indexed, vert_idxs = np.unique(verts_unindexed, return_inverse=True)

        index_size = 2 if len(verts_indexed) <= 65535 else 4

        # TODO: check whether indices are actually a benefit

        part_triangles = seek_align(blob, 4)
        write_ndarray(blob, vert_idxs.astype(f'<u{index_size}'))

        verts = verts_indexed

        part_positions = seek_align(blob, 4)
        write_ndarray(blob, verts['position'])

        part_normals = seek_align(blob, 4)
        write_ndarray(blob, verts['normal'])

        part_attributes = []

        part_attributes.append(seek_align(blob, 4))
        write_ndarray(blob, verts['UVMap'])

        parts.append(Part(
            Positions=part_positions,
            Normals=part_normals,
            Attributes=part_attributes,
            VertexCount=len(verts),
            IndexType={2: 'UINT16', 4: 'UINT32'}[index_size], # TODO: factor this map out pls
            Triangles=part_triangles,
            TriangleCount=len(tris),
        ))

    with open(directory, 'wb') as f:
        f.write(b'Worldspawn')
        seek_align(f, 16)
        f.write(b'Geometry')
        seek_align(f, 16)

        sections = f.tell()
        f.write(struct.pack('<qqqq', 0, 0, 0, 0))

        json_offset = f.seek(0, 1)
        h = Header(None, collider, parts)
        d = dataclasses.asdict(h, dict_factory=dict_skip_nulls)
        d = fixupdict(d)
        json.dump(d, UTF8Writer(f), default=util.asdasd)
        json_length = f.seek(0, 1) - json_offset

        blob_offset = f.seek(0, 1)
        f.write(blob.bytes())
        blob_length = f.seek(0, 1) - blob_offset

        f.seek(sections)
        f.write(struct.pack('<qqqq', json_offset, json_length, blob_offset, blob_length))

# TODO: move to util somewhere
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


# TODO: move these into utils or something, other cookers might also want to use
# this.
def write_ndarray(f, a):
    return f.write(a.tobytes())


# TODO: move into util or something like that package
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
