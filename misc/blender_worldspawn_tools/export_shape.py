import dataclasses
import json

import numpy as np

import bpy
from mathutils import Vector, Quaternion

from .util import asdasd, np_array_from_bpy_collection


# TODO: merge into export_geometry


@dataclasses.dataclass
class Shape:
    Kind: str


@dataclasses.dataclass
class Leaf(Shape):
    # % of volume filled
    MaterialIndex: int


@dataclasses.dataclass
class Sphere(Leaf):
    Radius: float


@dataclasses.dataclass
class Box(Leaf):
    HalfExtent: Vector


@dataclasses.dataclass
class Cylinder(Leaf):
    Radius: float
    Height: float


# TODO: can we move definition inside Mesh?
@dataclasses.dataclass
class Triangle:
    VertexIndices: list[int]
    MaterialIndex: int


# MaterialIndex will be the fall back material (when interpreted as ConvexHull)
@dataclasses.dataclass
class Mesh(Leaf):
    Vertices: list[Vector] # TODO: replace with an offset into the blob
    Triangles: list[Triangle] # TODO: replace with an offset into the blob


# TODO: maybe get rid of Transformed and Compound shapes, and just have a flat
# list at the root, kinda like PhysX would consume it?


# @dataclasses.dataclass
# class Transformed(Shape):
#     Translation: Vector
#     Rotation: Quaternion
#     Scale: Vector
#     Shape: Shape


# @dataclasses.dataclass
# class Compound(Shape):
#     Shapes: list[Shape]


# TODO: rename into something else
@dataclasses.dataclass
class Header2:
    Materials: list[str]
    PhysicsParts: Shape


# TODO: thread a "builder" object through f

def f(obj, depsgraph):
    rb = obj.rigid_body
    match rb.collision_shape:
        # case 'SPHERE':
        #     pass

        case 'BOX':
            return Box(Kind='Box', MaterialIndex=0, HalfExtent=obj.dimensions / 2)

        # case 'CYLINDER':
        #     pass

        # TODO: split these apart
        case 'CONVEX_HULL' | 'MESH':
            # which modifiers (all, deform, none) are applied to the collision mesh in blender depends on collision source setting
            me = obj.evaluated_get(depsgraph).to_mesh()
            me.calc_loop_triangles()
            vertices = np_array_from_bpy_collection(me.vertices, 'co', dtype=(np.float32, 3))
            indices = np_array_from_bpy_collection(me.loop_triangles, 'vertices', dtype=(np.uint32, 3))
            tri_materials = np_array_from_bpy_collection(me.loop_triangles, 'material_index', dtype=np.uint32)
            # print(np.c_[vertices, np.zeros(len(vertices))])
            triangles = np.c_[
                np_array_from_bpy_collection(me.loop_triangles, 'vertices', dtype=(np.uint32, 3)),
                np_array_from_bpy_collection(me.loop_triangles, 'material_index', dtype=np.uint32),
            ]

            kind = 'ConvexHull' if rb.collision_shape == 'CONVEX_HULL' else 'Mesh'
            return Mesh(
                Kind=kind,
                MaterialIndex=0,
                Vertices=[Vector(v) for v in vertices],
                Triangles=[Triangle(VertexIndices=[int(tri[0]), int(tri[1]), int(tri[2])], MaterialIndex=int(tri[3])) for tri in triangles],
            )

        # case 'COMPOUND':
        #     shapes = []
        #     for child in obj.children:
        #         if not child.rigid_body:
        #             continue
        #         shape = f(child)
        #         t, r, s = child.matrix_local.decompose()
        #         shapes.append(Transformed(Kind='Transformed', Translation=t, Rotation=r, Scale=s, Shape=shape))
        #     return Compound(Kind='Compound', Shapes=shapes)

        case _:
            assert False, f"{rb.collision_shape} not implemented"


def postprocess(shape):
    return shape


def validate(shape):
    pass


# TODO: _export should be just the code we can reuse in prefab/map export
def save(operator, context, obj):
    shape = f(obj, context.evaluated_depsgraph_get())

    # validate(shape)

    return json.dumps(dataclasses.asdict(Header2(Materials=[], PhysicsParts=shape)), indent='\t', default=asdasd).encode('utf-8')
