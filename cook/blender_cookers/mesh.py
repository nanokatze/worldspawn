import bpy
import dataclasses
import numpy as np

from cookers import mesh as mesh_cooker2

import numpyutil as nputil
import bpyutil


# TODO: should this be called an object cooker instead?


def deps(context, obj, dset):
    from blender_cookers import material as material_cooker

    dset.add_product((context.path_for_datablock(obj), 'Object', obj.name))

    for material_slot in obj.material_slots:
        material_cooker.deps(context, material_slot.material, dset)


def __triangulate(mesh):
    import bmesh
    bm = bmesh.new()
    bm.from_mesh(mesh)
    bmesh.ops.triangulate(bm, faces=bm.faces[:])
    bm.to_mesh(mesh)
    bm.free()


def cook(context, obj):
    materials = [None] * max(len(obj.material_slots), 1)
    for i, slot in enumerate(obj.material_slots):
        # TODO: should we produce a diagnostic when material is None?
        if slot.material is not None:
            materials[i] = context.path_for_datablock(slot.material)

    mesh = obj.to_mesh()

    # TODO: avoid generating tangents and get tangents the same way cycles does?
    try:
        mesh.calc_tangents()
    except Exception:
        __triangulate(mesh)
        mesh.calc_tangents()

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
    loop_vert_idxs = bpyutil.array_from_prop_collection(mesh.loops, 'vertex_index', dtype=np.uint32)

    # TODO: we don't ever manipulate these so we can always just use uint
    # vectors actually
    fields = []
    # TODO: also stick a material_index in here? If we're going to be exporting
    # custom user prims and they're gonna be per-primitive, we'll need to solve
    # fanning out material_index
    fields.append(('position', nputil.vec3))
    fields.append(('normal', nputil.vec3))
    # fields.append(('tangent', nputil.vec3)) # TODO: should be vec4 (x, y, z, sign)
    # group stuff here
    # user defined attrs here
    fields.append(('UVMap', nputil.vec2))

    loops = np.empty(len(mesh.loops), dtype=np.dtype(fields))

    loops['position'] = bpyutil.array_from_prop_collection(mesh.vertices, 'co', dtype=nputil.vec3)[loop_vert_idxs]

    # TODO: encode octahedrally
    loops['normal'] = bpyutil.array_from_prop_collection(mesh.loops, 'normal', dtype=nputil.vec3)

    # TODO: encode in a smarter way
    # TODO: are these the tangents we need? Does this match cycles' tangents
    # derived from ATTR_STD_GENERATED? There's also tangent spaces necessary for
    # normal mapping, which are computed from UV.
    # loops['tangent'] = bpyutil.array_from_prop_collection(mesh.loops, 'tangent', dtype=nputil.vec3)

    # TODO: user attribs here
    loops['UVMap'] = bpyutil.array_from_prop_collection(mesh.uv_layers['UVMap'].uv, 'vector', dtype=nputil.vec2)

    tri_loop_idxs = bpyutil.array_from_prop_collection(mesh.loop_triangles, 'loops', dtype=(np.uint32, 3))
    tris = loops[tri_loop_idxs]

    tri_mat_idxs = bpyutil.array_from_prop_collection(mesh.loop_triangles, 'material_index', dtype=np.uint32)

    # TODO: fold tri_mat_idxs into tris

    mesh_cooker2.cook(mesh_cooker2.Raw(materials, tris, tri_mat_idxs), context.path_for_datablock(obj))
