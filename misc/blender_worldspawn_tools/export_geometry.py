import dataclasses
import json
import struct
import time

import numpy as np

from mathutils import Vector

from .util import asdasd, fixupdict, dict_skip_nulls, np_array_from_bpy_collection, np_vec2, np_vec3

from . import export_shape

# TODO: LODs


def _make_vertex_dtype(vertex_group_elements_per_vertex, uv_layers):
    fields = [
        ('position', np_vec3),
        ('normal', np_vec3),
        ('group_indices', (np.uint32, (vertex_group_elements_per_vertex,))), # TODO: interleave these into a single attribute stream
        ('group_weights', (np.float32, (vertex_group_elements_per_vertex,))),
        *((f'uv{i}', np_vec2) for i in range(uv_layers)),
    ]
    return np.dtype(fields)


# TODO: we need two "Part" types, one would be the one we shove into the file,
# the other would have data associated with it


@dataclasses.dataclass
class Part:
    VertexPositions: int
    VertexNormals: int
    VertexGroupIndices: int
    VertexGroupWeights: int
    # TODO: this should be a string -> int map probably? Different parts might
    # have different sets of attributes. Though from blender's perspective this
    # isn't particularly sensible.
    VertexAttributes: list[int]
    VertexCount: int

    IndexType: str
    Indices: int
    PrimitiveCount: int

    # TODO: Material


# TODO: rename
@dataclasses.dataclass
class AttributeDesc:
    Name: str
    Type: str


# TODO: we need to share materials between detailed parts and simplified physics
# shapes.
@dataclasses.dataclass
class RenderingGeometry:
    VertexGroups: list[str]
    # TODO: might be a good idea to make this per-Part? Or perhaps even
    # different vertices should specify different bone count (with a single
    # byte.)
    VertexGroupElementsPerVertex: int
    VertexAttributes: list[AttributeDesc] # TODO: rename this field to something else? E.g. VertexAttributeNames or Descs or w/e
    Parts: list[Part] # TODO: rename to something else


# TODO: rename to just Header
@dataclasses.dataclass
class Header2:
    Type: str # TODO: do we need this? We share the general container with animations and armatures also so I guess we do
    Rendering: RenderingGeometry
    RigidBody: int


def gather(context, obj):
    depsgraph = context.evaluated_depsgraph_get()
    # TODO: handle this failing and report it like we do with calc_tangents
    me = obj.evaluated_get(depsgraph).to_mesh()

    vertex_groups = [vertex_group.name for vertex_group in obj.vertex_groups]

    max_group_elements_per_vertex = max(len(vert.groups) for vert in me.vertices) if obj.vertex_groups else 0

    uv_maps = [uv_layer.name for uv_layer in me.uv_layers]

    materials = []

    # try:
    #     me.calc_tangents()
    # except Exception as e:
    #     print(f'calc_tangents {obj}: {e}')
    #     # TODO: use operator.report to report this as an error
    #     # assert False, f'calc_tangents {obj}: {e}'

    me.calc_loop_triangles()

    vertex_dtype = _make_vertex_dtype(max_group_elements_per_vertex, len(me.uv_layers))

    tri_mat_idxs = np.zeros(len(me.loop_triangles), dtype=np.uint32) # np_array_from_bpy_collection(me.loop_triangles, 'material_index', dtype=np.uint32)

    loops = np.empty(len(me.loops), dtype=vertex_dtype)

    loop_vert_idxs = np_array_from_bpy_collection(me.loops, 'vertex_index', dtype=np.uint32)

    loops['position'] = np_array_from_bpy_collection(me.vertices, 'co', dtype=np_vec3)[loop_vert_idxs]
    loops['normal'] = np_array_from_bpy_collection(me.loops, 'normal', dtype=np_vec3)

    if obj.vertex_groups:
        indices = np.zeros(len(me.vertices), dtype=(np.uint32, (max_group_elements_per_vertex,)))
        weights = np.zeros(len(me.vertices), dtype=(np.float32, (max_group_elements_per_vertex,)))
        for vert in me.vertices:
            # BUG: Blender raises a RuntimeError if we foreach_get into an
            # oversized array. This does not happen with other collections. Is
            # this a bug?
            group_element_count = len(vert.groups)
            vert.groups.foreach_get('group', indices[vert.index][:group_element_count])
            vert.groups.foreach_get('weight', weights[vert.index][:group_element_count])

        # Vertex group elements must be in order of descending weight

        weights_descending = np.fliplr(np.argsort(weights, axis=1))

        loops['group_indices'] = np.take_along_axis(indices, weights_descending, axis=1)[loop_vert_idxs]
        loops['group_weights'] = np.take_along_axis(weights, weights_descending, axis=1)[loop_vert_idxs]

    for i, uv_layer in enumerate(me.uv_layers):
        loops[f'uv{i}'] = np_array_from_bpy_collection(uv_layer.uv, 'vector', dtype=np_vec2)
        # TODO: rewrite into matrix multiplication
        # TODO: actually, just fix up UVs in the shader
        loops[f'uv{i}'][::,1] *= -1
        loops[f'uv{i}'][::,1] += 1

    for i, attribute in enumerate(me.attributes):
        # print(i, attribute.name)
        pass

    tri_loop_idxs = np_array_from_bpy_collection(me.loop_triangles, 'loops', dtype=(np.uint32, 3))
    tri_loops = loops[tri_loop_idxs.flat].reshape((-1, 3))

    # TODO: we really want named tuples here

    return vertex_groups, max_group_elements_per_vertex, uv_maps, materials, tri_loops, tri_mat_idxs


# TODO: rename and also ensure alignment. Rename to e.g. list_extend_offset?
def extend_and_get_offset(buf, blob, align=1):
    off = len(buf)
    buf.extend(blob)
    return off


# TODO: rename, make nicer, etc
def pack_blob(buf, header_offset, blob):
    struct.pack_into('<qq', buf, header_offset, len(buf), len(blob))
    buf.extend(blob)


# TODO: just have this take the destination filename directly, or at least a
# function which we will poke once and it will get us the buffer
#
# TODO: ugh, this should not take operator but a weaker thing for diagnostics
# only.
def save(operator, context, obj, export_physics=False):
    t0 = time.monotonic()

    # TODO: fold gather with serialization into building the blob into one

    # TODO: rename this
    physics_shape = None
    if export_physics and obj.rigid_body:
        depsgraph = context.evaluated_depsgraph_get()

        physics_shape = export_shape.f(obj, depsgraph)

    vertex_groups, vertex_group_elements_per_vertex, uv_maps, materials, tri_loops, tri_mat_idxs = gather(context, obj)

    t1 = time.monotonic()

    flat_tri_loops = tri_loops.flat
    flat_tri_mat_idxs = np.repeat(tri_mat_idxs, 3)

    parts = [] # TODO: rename to serialized_parts or something along those lines?
    blob_data = bytearray()
    for material_index in np.unique(tri_mat_idxs):
        # TODO: building a more efficient representation should be done by
        # another function and not writeout?
        #
        # TODO: split by material *before* we gather stuff into a huge flat
        # vertex array, I think?
        vertices, indices = np.unique(flat_tri_loops[flat_tri_mat_idxs == material_index], return_inverse=True)

        # TODO: add an option for no indices? This should be useful for when every individual triangle
        # is animated and thus disjoint

        # TODO: and use correct endianness

        # TODO: don't export empty buffers
        positions_offset = extend_and_get_offset(blob_data, vertices['position'].tobytes())
        normals_offset = extend_and_get_offset(blob_data, vertices['normal'].tobytes())
        group_indices_offset = extend_and_get_offset(blob_data, vertices['group_indices'].tobytes())
        group_weights_offset = extend_and_get_offset(blob_data, vertices['group_weights'].tobytes())
        uvs_offsets = []
        for i in range(len(uv_maps)):
            uvs_offsets.append(extend_and_get_offset(blob_data, vertices[f'uv{i}'].tobytes()))
        vertex_count = len(vertices)

        index_size = 2 if len(vertices) <= 65535 else 4
        indices_offset = extend_and_get_offset(blob_data, indices.astype(f'<u{index_size}').tobytes())
        assert len(flat_tri_loops) % 3 == 0
        primitive_count = len(flat_tri_loops) // 3

        # TODO: move this translation top the top level
        index_type = {2: "UINT16", 4: "UINT32"}[index_size]

        parts.append(Part(
            VertexPositions=positions_offset,
            VertexNormals=normals_offset,
            VertexGroupIndices=group_indices_offset,
            VertexGroupWeights=group_weights_offset,
            VertexAttributes=uvs_offsets,
            VertexCount=vertex_count,
            IndexType=index_type,
            Indices=indices_offset,
            PrimitiveCount=primitive_count,
        ))

    t2 = time.monotonic()

    # TODO: write out the entire header at the end?

    buf = bytearray(48)
    struct.pack_into('16s', buf, 0, b'Worldspawn')

    header2 = Header2(
        Type='Geometry',
        Rendering=RenderingGeometry(
            VertexGroups=vertex_groups,
            VertexGroupElementsPerVertex=vertex_group_elements_per_vertex,
            VertexAttributes=[AttributeDesc(Name=uv_map, Type='R32G32_SFLOAT') for uv_map in uv_maps],
            Parts=parts,
        ),
        RigidBody=physics_shape,
    )

    # TODO: we need to fix up the resulting dict such as convert Quaternion and
    # Vector to lists of floats, and in turn convert floats to JS strings
    d = dataclasses.asdict(header2, dict_factory=dict_skip_nulls)
    d = fixupdict(d)
    pack_blob(buf, 16, json.dumps(d, default=asdasd).encode('utf-8'))
    pack_blob(buf, 32, blob_data)

    t3 = time.monotonic()

    print(f'gather={t1-t0}\n' +
        f'serialize={t2-t1}\n' +
        f'layout={t3-t2}\n' +
        f'total={t3-t0}')

    return buf
