import bpy
import dataclasses
import numpy as np

from cookers import mesh as mesh_cooker2

import numpyutil as nputil
import bpyutil


# TODO: should this be called an object cooker instead?


def deps(context, obj, dset):
    from blender_cookers import material as material_cooker

    # print('cooking', obj.name)

    dset.add_product((context.path_for_datablock(obj), 'Object', obj.name))

    # TODO: switch to iterating over obj.to_mesh().materials
    for material_slot in obj.material_slots:
        if material_slot.material:
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

    tri_corners_idxs = bpyutil.array_from_prop_collection(mesh.loop_triangles, 'loops', dtype=(np.uint32, 3))

    corner_vert_idxs = bpyutil.array_from_prop_collection(mesh.loops, 'vertex_index', dtype=np.uint32)

    tri_vert_idxs = corner_vert_idxs[tri_corners_idxs]

    positions = mesh_cooker2.AttributeBuffer(
        mesh_cooker2.Domain.VERTEX,
        bpyutil.array_from_prop_collection(mesh.vertices, 'co', dtype=nputil.vec3)[tri_vert_idxs])

    # TODO: encode octahedrally?
    normals = mesh_cooker2.AttributeBuffer(
        mesh_cooker2.Domain.VERTEX,
        bpyutil.array_from_prop_collection(mesh.loops, 'normal', dtype=nputil.vec3)[tri_corners_idxs])

    # TODO: tangents

    joints = [g.name for g in obj.vertex_groups]

    # joint_weights = [None] * len(mesh.vertices)
    # for i, v in enumerate(mesh.vertices):
    #     joint_weights[i] = [(np.uint32(g.group), np.float32(g.weight)) for g in v.groups], dtype=np.dtype([('index', np.uint32), ('weight', np.float32)])

    # print(tri_vert_idxs)
    # joint_weights2 = joint_weights[tri_vert_idxs]

    # print(joint_weights2.shape, joint_weights2.dtype)
    # print(joint_weights2[np.array([1, 0])])

    # TODO: materials can come from the mesh itself actually
    materials = [None] * max(len(obj.material_slots), 1)
    for i, slot in enumerate(obj.material_slots):
        # TODO: should we produce a diagnostic when material is None?
        if slot.material is not None:
            materials[i] = context.path_for_datablock(slot.material)

    material_indices = mesh_cooker2.AttributeBuffer(
        mesh_cooker2.Domain.FACE,
        bpyutil.array_from_prop_collection(mesh.loop_triangles, 'material_index', dtype=np.uint32))

    # for attr in mesh.attributes:
    #     # Already handled
    #     if attr.name == 'position':
    #         continue

    #     # TODO: skip other things? Also when verbose mode is on we should print
    #     # why we skip something
    #     if attr.name.startswith("."):
    #         continue

    #     # TODO: we should be able to handle all blender data types, ackshually.
    #     if attr.data_type not in _ATTRIBUTE_TYPE_TO_NUMPY_TYPE:
    #         continue

    #     dt = _ATTRIBUTE_TYPE_TO_NUMPY_TYPE[attr.data_type]

    #     # TODO: also handle face domain and think of what to do with edge
    #     match attr.domain:
    #         case 'POINT':
    #             domain = mesh_cooker2.Domain.VERTEX
    #             data = bpyutil.array_from_prop_collection(attr.data, 'vector', dt)[tri_vert_idxs]
    #         case 'CORNER':
    #             domain = mesh_cooker2.Domain.VERTEX
    #             data = bpyutil.array_from_prop_collection(attr.data, 'vector', dt)[tri_corners_idxs]
    #         case _:
    #             continue
    #     attr_buffers.append(mesh_cooker2.AttributeBuffer(domain, data))
    #     attributes[attr.name] = len(attr_buffers)-1

    mesh2 = mesh_cooker2.Raw(
        positions=positions,
        normals=normals,
        joints=joints,
        joint_weights=None,
        materials=materials,
        material_indices=material_indices,
        named_attributes={})

    mesh_cooker2.cook(mesh2, context.path_for_datablock(obj))
