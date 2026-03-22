# SPDX-License-Identifier: Apache-2.0
# Copyright 2024-2025 Caterina Shablia

# TODO: rename this addon from blender worldspawn tools to worldspawn integration
bl_info = {
    "name": "Worldspawn Integration",
    "blender": (4, 5, 0),
    "author": "Caterina Shablia",
    "category": "???", # TODO: what do we plop here
}


if "bpy" in locals():
    import importlib


# TODO: rename "Export" to "Cook"?


import bpy
from bpy.props import BoolProperty, StringProperty, PointerProperty

from . import blender_schema


class ACTION_PT_worldspawn(bpy.types.Panel):
    bl_space_type = 'DOPESHEET_EDITOR'
    bl_region_type = 'UI'
    bl_label = 'Worldspawn'
    bl_category = 'Action'

    @classmethod
    def poll(self, context):
        return context.active_action is not None

    def draw(self, context):
        action = context.active_action

        layout = self.layout
        layout.prop(action.worldspawn, 'export')


class COLLECTION_PT_worldspawn(bpy.types.Panel):
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_label = 'Worldspawn'
    bl_context = 'collection'

    def draw(self, context):
        collection = context.collection

        layout = self.layout
        layout.prop(collection.worldspawn, 'export')


class OBJECT_PT_worldspawn(bpy.types.Panel):
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_label = 'Worldspawn'
    bl_context = 'object'

    @classmethod
    def poll(self, context):
        return context.object is not None

    def draw(self, context):
        obj = context.object

        layout = self.layout
        layout.prop(obj.worldspawn, 'export')
        layout.prop(obj.worldspawn, 'values')


class SCENE_PT_worldspawn(bpy.types.Panel):
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_label = 'Worldspawn'
    bl_context = 'scene'

    def draw(self, context):
        scene = context.scene

        layout = self.layout
        layout.prop(scene.worldspawn, 'export')
        layout.prop(scene.worldspawn, 'values')


__classes = [
    ACTION_PT_worldspawn,
    COLLECTION_PT_worldspawn,
    OBJECT_PT_worldspawn,
    SCENE_PT_worldspawn,
]


def register():
    blender_schema.register_classes()

    for cls in __classes:
        bpy.utils.register_class(cls)


def unregister():
    for cls in __classes:
        bpy.utils.unregister_class(cls)

    blender_schema.unregister_classes()
