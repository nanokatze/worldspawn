import bpy
import json
from mathutils import Vector, Quaternion

import util


def deps(context, scene, dset):
    import mesh_cooker

    dset.add_product((context.path_for_datablock(scene), 'Scene', scene.name))

    for obj in scene.objects:
        if obj.hide_render:
            continue

        mesh_cooker.deps(context, obj, dset)


def cook(context, scene):
    # TODO: consume a schema file for this
    cooked_scene = {
        'Sky': 'skies/industrial_sunset_puresky.ktx2', # TODO: output path to the sky material, or emit the sky material as-is
        'Gravity': Vector((0, 0, -10)), # TODO: let the user configure this
        'TranslationRotation': {},
        'Scale': {},
        'PhysicsShape': {},
        'PhysicsLayer': {},
        'PhysicsMotionType': {},
        'RendererModel': {},
        'PlayerSpawn': {},
    }

    entity = 1
    for obj in scene.objects:
        # TODO: explain why this is a good thing to filter on?
        if obj.hide_render:
            continue

        comps_bpy = obj.get('worldspawn.components')
        comps = comps_bpy.to_dict() if comps_bpy else {}

        T, R, S = obj.matrix_world.decompose()
        # TODO: should we always overwrite these?
        # We might want to warn or error if these comps are already set.
        # Or don't overwrite if these are already set.
        # Erroring out seems to be the more useful option of the two.
        comps['TranslationRotation'] = {
            'Translation': T,
            'Rotation': R,
        }
        comps['Scale'] = S

        model_filename = context.path_for_datablock(obj)
        if model_filename:
            # TODO: do we need to do anything about this?
            if 'RendererModel' not in comps:
                comps['RendererModel'] = {
                    'Kind': 'FileBacked',
                    'Filename': model_filename,
                }

            if 'PhysicsShape' not in comps:
                # BUG: we should always export PhysicsShape, but whether the object
                # has any physics should be controlled at runtime by another
                # component, e.g. the physics layer. This means that yes, we'll want
                # to add another physics layer.
                if obj.rigid_body is not None:
                    comps['PhysicsShape'] = {
                        'Kind': 'FileBacked',
                        'Filename': model_filename,
                    }

        for comp, value in comps.items():
            cooked_scene[comp][entity] = value
        entity += 1

    with open(context.path_for_datablock(scene), 'wb') as f:
        f.write(json.dumps(cooked_scene, indent='\t', default=util.asdasd).encode('utf-8'))
