import bpy
from bpy.props import BoolProperty, StringProperty, PointerProperty


class WorldspawnActionSettings(bpy.types.PropertyGroup):
    export: BoolProperty(
        name='Export',
        description='Export this action',
    )

    @classmethod
    def register(cls):
        bpy.types.Action.worldspawn = bpy.props.PointerProperty(type=cls)

    @classmethod
    def unregister(cls):
        del bpy.types.Action.worldspawn


class WorldspawnCollectionSettings(bpy.types.PropertyGroup):
    export: BoolProperty(
        name='Export',
        description='Export this collection',
    )

    @classmethod
    def register(cls):
        bpy.types.Collection.worldspawn = bpy.props.PointerProperty(type=cls)

    @classmethod
    def unregister(cls):
        del bpy.types.Collection.worldspawn


class WorldspawnMaterialSettings(bpy.types.PropertyGroup):
    export: BoolProperty(
        name='Export',
        description='Export this material',
    )

    @classmethod
    def register(cls):
        bpy.types.Material.worldspawn = bpy.props.PointerProperty(type=cls)

    @classmethod
    def unregister(cls):
        del bpy.types.Material.worldspawn


class WorldspawnObjectSettings(bpy.types.PropertyGroup):
    # TODO: rename into export_geometry and add another such thing to the geometry data blocks
    export: BoolProperty(
        name='Export object data',
        description='Export this object\'s geometry',
    )
    values: StringProperty(name='Values (JSON)')

    @classmethod
    def register(cls):
        bpy.types.Object.worldspawn = bpy.props.PointerProperty(type=cls)

    @classmethod
    def unregister(cls):
        del bpy.types.Object.worldspawn


class WorldspawnSceneSettings(bpy.types.PropertyGroup):
    export: BoolProperty(
        name='Export',
        description='Export this scene',
    )
    values: StringProperty(name='Values (JSON)')

    @classmethod
    def register(cls):
        bpy.types.Scene.worldspawn = bpy.props.PointerProperty(type=cls)

    @classmethod
    def unregister(cls):
        del bpy.types.Scene.worldspawn


register_classes, unregister_classes = bpy.utils.register_classes_factory((
    WorldspawnActionSettings,
    WorldspawnCollectionSettings,
    WorldspawnMaterialSettings,
    WorldspawnObjectSettings,
    WorldspawnSceneSettings,
))
