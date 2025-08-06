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


import bpy
from bpy.props import BoolProperty, PointerProperty


# TODO: remove un favor of dynamic properties
class WorldspawnSceneComponents(bpy.types.PropertyGroup):
    Gravity: bpy.props.FloatVectorProperty(
        name='Gravity',
        description='',
        subtype='ACCELERATION',
    )


class WorldspawnSceneSettings(bpy.types.PropertyGroup):
    export: BoolProperty(
        name='Export',
        description='Export this scene',
    )
    components: PointerProperty(type=WorldspawnSceneComponents)

    @classmethod
    def register(cls):
        bpy.types.Scene.worldspawn = PointerProperty(type=cls)

    @classmethod
    def unregister(cls):
        del bpy.types.Scene.worldspawn


# TODO: should be initialized like a list or ordered map so that component order
# is good
_COMPONENTS = {
    'CollisionLayer': (0, ''),
    'PhysicsMassOverride': (1.0, ''),
    'PlayerSpawn': ({}, 'struct{}'),
}

# TODO: prefab support

# TODO: remove, add properties dynamically from _COMPONENTS
class WorldspawnObjectComponents(bpy.types.PropertyGroup):
    # TODO: how would we handle Entity component? We'd probs want it to be any
    # arbitrary python dict or w/e

    # TODO: make these store strings for compatibility
    CollisionLayer: bpy.props.EnumProperty(
        name='Collision Layer',
        description='',
        items=[
            ('NonMoving', 'Non-moving', ''),
            ('Moving', 'Moving', ''),
            ('Projectiles', 'Projectiles', ''),
        ],
    )

    PhysicsMassOverride: bpy.props.FloatProperty(
        name='Mass Override',
        min=0,
        max=1000000000,
    )


class WorldspawnObjectSettings(bpy.types.PropertyGroup):
    export: BoolProperty(
        name='Export object data',
        description='Export this object',
    )
    components: PointerProperty(type=WorldspawnObjectComponents)

    @classmethod
    def register(cls):
        bpy.types.Object.worldspawn = PointerProperty(type=cls)

    @classmethod
    def unregister(cls):
        del bpy.types.Object.worldspawn


class SubMenu(bpy.types.Menu):
    bl_idname = "OBJECT_MT_select_submenu"
    bl_label = "Select"

    def draw(self, context):
        obj = context.object
        comps = obj.worldspawn.components

        layout = self.layout

        for comp in _COMPONENTS:
            if comp in comps:
                continue
            layout.operator('worldspawn.add_component', text=comp).comp = comp


class IDK_AddComponent(bpy.types.Operator):
    bl_idname = "worldspawn.add_component"
    bl_label = "Add Component"

    comp: bpy.props.StringProperty()

    def invoke(self, context, event):
        bpy.ops.wm.call_menu(name="OBJECT_MT_select_submenu")
        return {'FINISHED'}

    def execute(self, context):
        obj = context.object
        comps = obj.worldspawn.components
        comps[self.comp] = _COMPONENTS[self.comp][0]
        return {'FINISHED'}


class IDK_DelComponent(bpy.types.Operator):
    bl_idname = "worldspawn.del_component"
    bl_label = "-"

    comp: bpy.props.StringProperty()

    def execute(self, context):
        obj = context.object
        comps = obj.worldspawn.components
        del comps[self.comp]
        return {'FINISHED'}


class SCENE_PT_worldspawn(bpy.types.Panel):
    bl_label = 'Worldspawn'
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_context = 'scene'

    def draw(self, context):
        scene = context.scene
        comps = scene.worldspawn.components

        layout = self.layout
        layout.prop(scene.worldspawn, 'export')

        panel_comps_header, panel_comps = layout.panel('comps')
        if panel_comps is not None:
            layout.prop(comps, 'Gravity')


class OBJECT_PT_worldspawn(bpy.types.Panel):
    bl_label = 'Worldspawn'
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_context = 'object'

    @classmethod
    def poll(self, context):
        return context.object is not None

    def draw(self, context):
        obj = context.object
        comps = obj.worldspawn.components

        layout = self.layout
        layout.prop(obj.worldspawn, 'export')
        # TODO: attributes to export
        # TODO: move components into their own panel?
        panel_comps_header, panel_comps = layout.panel('comps')
        panel_comps_header.label(text='Components')
        # TODO: make the experience more comparable to modifiers
        if panel_comps is not None:
            panel_comps.operator('worldspawn.add_component')
            for comp in _COMPONENTS:
                if comp not in comps:
                    continue
                cols = panel_comps.split(factor=0.1)
                cols.operator('worldspawn.del_component').comp = comp
                if _COMPONENTS[comp][1] == 'struct{}':
                    cols.label(text=comp)
                else:
                    cols.prop(comps, comp)


class WorldspawnMaterialSettings(bpy.types.PropertyGroup):
    export: BoolProperty(
        name='Export',
        description='Export this material',
    )

    @classmethod
    def register(cls):
        bpy.types.Material.worldspawn = PointerProperty(type=cls)

    @classmethod
    def unregister(cls):
        del bpy.types.Material.worldspawn

classes = [
    SubMenu,
    IDK_AddComponent,
    IDK_DelComponent,
    OBJECT_PT_worldspawn,
    SCENE_PT_worldspawn,
    # TODO: shove registration of components classes into registration of the
    # classes that refer to them? They have to be un/registered in a particular
    # order.
    WorldspawnMaterialSettings,
    WorldspawnObjectComponents,
    WorldspawnObjectSettings,
    WorldspawnSceneComponents,
    WorldspawnSceneSettings,
]


def register():
    for cls in classes:
        bpy.utils.register_class(cls)


def unregister():
    for cls in reversed(classes):
        bpy.utils.unregister_class(cls)
