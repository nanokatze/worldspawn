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


def __get(x, a):
    for p in a:
        if x is None:
            return None
        x = x.get(p)
    return x


def cook(context, scene):
    # TODO: consume a schema file for components

    cooked_scene = dict(scene.get('worldspawn', {}).get('components', {}))

    # TODO: output path to the sky material, or emit the sky material as-is
    cooked_scene['Sky'] = 'skies/industrial_sunset_puresky.ktx2'

    entity = 1
    for obj in scene.objects:
        # TODO: explain why this is a good thing to filter on?
        if obj.hide_render:
            continue

        comps = dict(obj.get('worldspawn', {}).get('components', {}))

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

        geometry = context.path_for_datablock(obj)
        if geometry:
            # TODO: do we need to do anything about this?
            if 'RenderingGeometry' not in comps:
                comps['RenderingGeometry'] = {
                    'Kind': 'FileBacked',
                    'Filename': geometry,
                }

            if 'CollisionGeometry' not in comps:
                comps['CollisionGeometry'] = {
                    'Kind': 'FileBacked',
                    'Filename': geometry,
                }

        for k, v in comps.items():
            if k not in cooked_scene:
                cooked_scene[k] = {}
            cooked_scene[k][entity] = v
        entity += 1

    cooked_scene = util.fixupdict(cooked_scene) # pain
    with open(context.path_for_datablock(scene), 'wb') as f:
        f.write(json.dumps(cooked_scene, indent='\t', default=util.asdasd).encode('utf-8'))
