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


_ATTRIBUTE_TYPE_TO_NUMPY_TYPE = {
    'FLOAT_VECTOR': np.dtype((np.float32, 3)),
    'FLOAT2': np.dtype((np.float32, 2)),
}


def cook(context, obj):
    materials = [None] * max(len(obj.material_slots), 1)
    for i, slot in enumerate(obj.material_slots):
        # TODO: should we produce a diagnostic when material is None?
        if slot.material is not None:
            materials[i] = context.path_for_datablock(slot.material)

    mesh = obj.to_mesh()

    # TODO: avoid generating tangents and get tangents the same way cycles does?
    # try:
    #     mesh.calc_tangents()
    # except:
    #     __triangulate(mesh)
    #     mesh.calc_tangents()

    __triangulate(mesh)

    mesh.calc_loop_triangles()

    # TODO: the user might want to specify quantization modes for varios
    # attributes (e.g. UNORM or FLOAT16 instead of the default FLOAT32.) We
    # could also support octahedral encoding for stuff like normals I guess.

    corner_vert_idxs = bpyutil.array_from_prop_collection(mesh.loops, 'vertex_index', dtype=np.uint32)

    tri_corners_idxs = bpyutil.array_from_prop_collection(mesh.loop_triangles, 'loops', dtype=(np.uint32, 3))

    # TODO: change stuff to be attr.name -> (attr.domain, attr.data) or similar
    attrs = {
        'material_index': mesh_cooker2.AttributeBuffer(mesh_cooker2.Domain.FACE, bpyutil.array_from_prop_collection(mesh.loop_triangles, 'material_index', dtype=np.uint32)),
    }

    for attr in mesh.attributes:
        # TODO: skip other things? Also when verbose mode is on we should print
        # why we skip something
        if attr.name.startswith("."):
            continue

        # TODO: we should be able to handle all blender data types, ackshually.
        if attr.data_type not in _ATTRIBUTE_TYPE_TO_NUMPY_TYPE:
            continue

        dt = _ATTRIBUTE_TYPE_TO_NUMPY_TYPE[attr.data_type]

        # TODO: also handle face domain and think of what to do with edge
        match attr.domain:
            case 'POINT':
                data = bpyutil.array_from_prop_collection(attr.data, 'vector', dt)
                attrs[attr.name] = mesh_cooker2.AttributeBuffer(mesh_cooker2.Domain.VERTEX, data[corner_vert_idxs][tri_corners_idxs])
            case 'CORNER':
                data = bpyutil.array_from_prop_collection(attr.data, 'vector', dt)
                attrs[attr.name] = mesh_cooker2.AttributeBuffer(mesh_cooker2.Domain.VERTEX, data[tri_corners_idxs])
            case _:
                continue

    # TODO: encode octahedrally?
    attrs['normal'] = mesh_cooker2.AttributeBuffer(mesh_cooker2.Domain.VERTEX, bpyutil.array_from_prop_collection(mesh.loops, 'normal', dtype=nputil.vec3)[tri_corners_idxs])

    # TODO: encode in a smarter way
    # TODO: are these the tangents we need? Does this match cycles' tangents
    # derived from ATTR_STD_GENERATED? There's also tangent spaces necessary for
    # normal mapping, which are computed from UV.
    # loops['tangent'] = bpyutil.array_from_prop_collection(mesh.loops, 'tangent', dtype=nputil.vec3)

    mesh_cooker2.cook(mesh_cooker2.Raw(materials, attrs), context.path_for_datablock(obj))
