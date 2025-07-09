# SPDX-License-Identifier: Apache-2.0
# Copyright 2018-2021 The glTF-Blender-IO authors.
# Copyright 2022-2024 Constantine Shablia

bl_info = {
    "name": "Worldspawn Tools",
    "blender": (3, 0, 0),
    "author": "Constantine Shablia",
    "category": "Import-Export",
}


def reload_package(module_dict_main):
    import importlib
    import pathlib

    def reload_package_recursive(current_dir, module_dict):
        for path in current_dir.iterdir():
            if "__init__" in str(path) or path.stem not in module_dict:
                continue

            if path.is_file() and path.suffix == ".py":
                importlib.reload(module_dict[path.stem])
            elif path.is_dir():
                reload_package_recursive(path, module_dict[path.stem].__dict__)

    reload_package_recursive(pathlib.Path(__file__).parent, module_dict_main)


if "bpy" in locals():
    reload_package(locals())


import dataclasses
import json
import os
import traceback

import bpy
from bpy.props import StringProperty, BoolProperty, EnumProperty, FloatProperty, FloatVectorProperty, IntProperty, IntVectorProperty
from bpy.types import Operator, Panel, PropertyGroup
from bpy_extras.io_utils import ImportHelper, ExportHelper
from mathutils import Vector, Quaternion

# from . import export_animation
from . import export_armature
from . import export_geometry
from . import export_material
from . import fs
from .util import asdasd


# TODO: unify object name mangling


# TODO: get this from env var, or find it automatically by looking for
# .projectroot file or smth
PROJECT_DIR = fs.DirFS("/home/nanokatze/code/worldspawn/raw")


# TODO: we should pass a function instead of outfile once we make sure this only
# exports one file.
def export_map(operator, context, scene, directory):
    # TODO: get Sky, Gravity out of the scene

    # TODO: meh? this will probably blow up on Windows so we should probably
    # convert windows path to unix in the operator. This is only used by code we
    # should move elsewhere anyway.
    # directory = '/'.join(outfile.split('/')[:-1])

    world = {
        'Sky': 'skies/industrial_sunset_puresky.ktx2',
        'Gravity': Vector((0, 0, -10)),
        'TranslationRotation': {},
        'Scale': {},
        'PhysicsShape': {},
        'PhysicsLayer': {},
        'PhysicsMotionType': {},
        'RendererModel': {},
        'PlayerSpawn': {},
    }

    next_entity = 1
    for obj in context.scene.objects:
        # TODO: the filtering criteria should probably be configurable somehow?
        #
        # Old note was: check boxes for limiting to selected (useful for small
        # prefabs?), visible (on by default, same behavior as BGE), renderable
        # objects
        if obj.hide_render:
            continue

        basename_shape = None
        basename_mesh = None

        # TODO: come up with a way to do external (file-backed) models
        match obj.type:
            case 'ARMATURE':
                # TODO: think whether we can export armatures here
                pass
            case 'LIGHT':
                # TODO: we don't have special light objects, this would be just
                # a spherical geometry with an appropriate material
                pass
            case _:
                # TODO: we should collect things into list of things to export
                # and let the Cook operator manage exporting, rather than export
                # by ourselves here.

                try:
                    blob = export_geometry.save(operator, context, obj, obj.rigid_body is not None)
                except Exception:
                    # TODO:
                    # https://docs.blender.org/api/current/bpy_types_enum_items/wm_report_items.html#rna-enum-wm-report-items
                    # self.report({'ERROR'}, 'TEST')
                    print(f'Exporting {obj.name} failed:')
                    print(traceback.format_exc().rstrip('\n'))
                else:
                    if blob:
                        # TODO: use subdirs to distinguish types instead?
                        # TODO: use Geometries_ prefix?
                        basename_mesh = 'Geometry_' + bpy.path.clean_name(obj.name)
                        if obj.rigid_body is not None:
                            basename_shape = basename_mesh
                        with open(os.path.join(directory, basename_mesh), 'wb') as f:
                            f.write(blob)

        entity = next_entity
        next_entity += 1

        # TODO: remove worldspawn_components support when we migrate all of our
        # blends
        components = obj.get('worldspawn.components') or obj.get('worldspawn_components')
        if components:
            components = components.to_dict()
        else:
            components = {}

        t, r, s = obj.matrix_world.decompose()
        components['TranslationRotation'] = {'Translation': t, 'Rotation': r}
        components['Scale'] = s

        # TODO: rename to PhysicsGeometry
        if basename_shape and 'PhysicsShape' not in components:
            components['PhysicsShape'] = {'Kind': 'FileBacked', 'Filename': 'maps/lockdown/' + basename_shape}

        # TODO: rename this component to RenderingGeometry or something
        if basename_mesh and 'RendererModel' not in components:
            components['RendererModel'] = {'Kind': 'FileBacked', 'Filename': 'maps/lockdown/' + basename_mesh}

        for k, v in components.items():
            world[k][entity] = v

    # TODO: use a proper join operator
    with open(directory + '/Scenes_' + scene.name, 'wb') as f:
        f.write(json.dumps(world, indent='\t', default=asdasd).encode('utf-8'))


class Cook(Operator):
    """This appears in the tooltip of the operator and in the generated docs"""
    bl_idname = 'worldspawn_tools.cook'
    bl_label = 'Export Assets for Worldspawn'


    directory: StringProperty(
        name='Dir Path',
        description='Directory used for exporting the cooked assets',
        maxlen=4096,
    )


    def invoke(self, context, _event):
        self.directory = os.path.dirname(context.blend_data.filepath or PROJECT_DIR.dir)
        context.window_manager.fileselect_add(self)
        return {'RUNNING_MODAL'}


    def execute(self, context):
        # TODO: walk over datablocks we're interested in: materials,
        # objects/object data (geometry), scenes (in that order) and export
        # them. We should automatically discover dependent things that should be
        # exported (e.g. materials) by default, but export only specific
        # datablocks when provided that list (which our autocooker would do.)

        blend_data = bpy.context.blend_data

        for material in blend_data.materials:
            if material.library:
                continue # don't export linked stuff
            if not material.get('worldspawn.export'):
                continue
            # export_material.test(material)

        # return {'FINISHED'}

        for object in blend_data.objects:
            if material.library:
                continue
            if not object.get('worldspawn.export'):
                continue
            match object.type:
                case 'ARMATURE':
                    export_armature.save(self, context, object, self.directory + '/Armatures_' + object.name)
                case 'MESH':
                    with open(self.directory + '/Geometry_' + object.name, 'wb') as f:
                        f.write(export_geometry.save(self, context, object, object.rigid_body is not None))
                case _:
                    # TODO: this should be an error
                    print(f'unsupported object type {object.type}')

        for scene in blend_data.scenes:
            if not scene.get('worldspawn.export'):
                continue
            export_map(self, context, scene, self.directory)

        return {'FINISHED'}


class OBJECT_PT_worldspawn(Panel):
    bl_label = 'Worldspawn'
    bl_space_type = 'PROPERTIES'
    bl_region_type = 'WINDOW'
    bl_context = 'object'

    def draw(self, context):
        # TODO: print a warning/error that we're not in worldspawn project root
        row = self.layout.row()
        row.label(text='TODO')


EXPORT_OPERATORS = [Cook]


def menu_func_file_export(self, context):
    # TODO: Ugh, why do we do lstrip here? check what other addons are doing.
    for cls in EXPORT_OPERATORS:
        self.layout.operator(cls.bl_idname, text=cls.bl_label.lstrip('Export '))


classes = [
    *EXPORT_OPERATORS,
    OBJECT_PT_worldspawn,
]


def register():
    for cls in classes:
        bpy.utils.register_class(cls)

    bpy.types.TOPBAR_MT_file_export.append(menu_func_file_export)


def unregister():
    for cls in classes:
        bpy.utils.unregister_class(cls)

    bpy.types.TOPBAR_MT_file_export.remove(menu_func_file_export)
