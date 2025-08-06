import bpy
import dataclasses
import numpy as np

import mesh_cooker2
import numpy_util as nputil


# TODO: rename to object_cooker?


def deps(context, object, dset):
    dset.add_product((context.path_for_datablock(object), 'Object', object.name))


def cook(context, object):
    mesh = object.to_mesh()

    mesh.calc_loop_triangles()

    # TODO: use "corners" instead of "loops"? "loops" is a horrible name

    # blender_attr_type_to_np_type = {
    #     'FLOAT_VECTOR': nputil.vec3,
    #     'FLOAT2': nputil.vec2,
    # }

    # TODO: the user might want to specify quantization modes for varios
    # attributes (e.g. UNORM or FLOAT16 instead of the default FLOAT32.)

    # print(object.get('worldspawn.export_attributes'))

    # vertices -> loops permutation
    loop_vert_idxs = nputil.array_from_bpy_collection(mesh.loops, 'vertex_index', dtype=np.uint32)

    # TODO: we don't ever manipulate these so we can always just use uint
    # vectors actually
    fields = []
    # TODO: also stick a material_index in here? If we're going to be exporting
    # custom user prims and they're gonna be per-primitive, we'll need to solve
    # fanning out material_index
    fields.append(('position', nputil.vec3))
    fields.append(('normal', nputil.vec3))
    # group stuff here
    # user defined attrs here
    # TODO: prefix user attrs? e.g. with "attributes."
    fields.append(('UVMap', nputil.vec2))

    loops = np.empty(len(mesh.loops), dtype=np.dtype(fields))

    loops['position'] = nputil.array_from_bpy_collection(mesh.vertices, 'co', dtype=nputil.vec3)[loop_vert_idxs]

    # TODO: encode octahedrally
    loops['normal'] = nputil.array_from_bpy_collection(mesh.loops, 'normal', dtype=nputil.vec3)

    # TODO: user attribs here
    loops['UVMap'] = nputil.array_from_bpy_collection(mesh.uv_layers['UVMap'].uv, 'vector', dtype=nputil.vec2)

    tri_loop_idxs = nputil.array_from_bpy_collection(mesh.loop_triangles, 'loops', dtype=(np.uint32, 3))
    tris = loops[tri_loop_idxs]

    tri_mat_idxs = nputil.array_from_bpy_collection(mesh.loop_triangles, 'material_index', dtype=np.uint32)

    mesh_cooker2.cook(mesh_cooker2.Raw(tris, tri_mat_idxs), context.path_for_datablock(object))
