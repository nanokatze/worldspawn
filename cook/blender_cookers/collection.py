import json

from mathutils import Matrix

import util
import bpyutil


def __should_cook_object(obj):
    if obj.hide_render:
        return False
    return True


def __should_cook_object_data(obj):
    if obj.type not in {'MESH', 'FONT'}:
        return False
    return True


def deps(context, collection, dset):
    from blender_cookers import mesh as mesh_cooker

    for child in collection.children:
        if collection.hide_render:
            continue
        deps(context, child, dset)

    for obj in collection.objects:
        if not __should_cook_object(obj):
            continue
        # TODO: diagnose when there's objects we don't know how to handle? e.g.
        # LIGHT
        if not __should_cook_object_data(obj):
            continue
        mesh_cooker.deps(context, obj, dset)

    # HACK: we should skip exporting scene.collection (it's special and doesn't
    # appear in the collection datablocks) but not in this horrible way
    if collection.name != 'Scene Collection':
        dset.add_product((context.path_for_datablock(collection), 'Collection', collection.name))


# TODO: don't duplicate this,,,
class __Cooker:


    def add_entity(self, comps):
        cooked = self.cooked
        for k, v in comps.items():
            if k not in cooked:
                cooked[k] = {}
            cooked[k][self.entity] = v
        self.entity += 1


# TODO: make this take objects rather than the entire collection
def cook_objects_into(context, xform, collection, cooked_scene):
    for obj in collection.objects:
        if not __should_cook_object(obj):
            continue

        comps = dict(obj.get('worldspawn', {}).get('components', {}))

        # TODO: should we always overwrite the components?
        # We might want to warn or error if these comps are already set. Or
        # don't overwrite if these are already set. Erroring out seems to be the
        # more useful option of the two.

        comps['Name'] = obj.name

        T, R, S = (xform @ obj.matrix_world).decompose()
        comps['LocalTranslationRotation'] = {
            'Translation': T,
            'Rotation': R,
        }
        comps['Scale'] = S

        if __should_cook_object_data(obj):
            geometry = context.path_for_datablock(obj)

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

        if obj.instance_collection is not None:
            comps['CollectionInstance'] = {
                'Filename': context.path_for_datablock(obj.instance_collection),
            }

        cooked_scene.add_entity(comps)


def cook(context, datablock):
    tmp = __Cooker()
    tmp.cooked = {}
    tmp.entity = 1

    __handle_collection(context, tmp, datablock, Matrix())

    cooked = bpyutil.fixupdict(tmp.cooked) # pain
    with open(context.path_for_datablock(datablock), 'wb') as f:
        json.dump(cooked, util.UTF8Writer(f), indent='\t', default=bpyutil.asdasd)



# TODO: in the future we'll need to go over objects two times: once to assign
# IDs and once more to actually collect
def __handle_collection(context, cooked_scene, collection, xform):
    for child_collection in collection.children:
        if child_collection.hide_render:
            continue
        __handle_collection(context, cooked_scene, child_collection, xform)

    cook_objects_into(context, xform, collection, cooked_scene)
