import bpy
import json
from mathutils import Matrix, Vector, Quaternion

import util
import bpyutil

from . import collection as collection_cooker


def deps(context, scene, dset):
    # import mesh_cooker

    dset.add_product((context.path_for_datablock(scene), 'Scene', scene.name))

    __handle_collection2(context, scene.collection, dset)


def __handle_collection2(context, collection, dset):
    for child_collection in collection.children:
        if child_collection.hide_render:
            continue
        __handle_collection2(context, child_collection, dset)

    # TODO: introduce deps_into for the purpose of skipping root scene?
    collection_cooker.deps(context, collection, dset)


# TODO: move this into cookers (not blender_cookers)
class __Cooker:


    def add_entity(self, comps):
        cooked = self.cooked
        for k, v in comps.items():
            if k not in cooked:
                cooked[k] = {}
            cooked[k][self.entity] = v
        self.entity += 1


# TODO: in the future we'll need to go over objects two times: once to assign
# IDs and once more to actually collect
def __handle_collection(context, cooked_scene, collection, xform):
    # Should we inline collections or instance them at 0x0x0? I guess both,
    # inline if they're not excluded from view layer and also instance? Only
    # ever instancing would be pretty elegant, but that means refs from inside
    # blender can't be done. Actually they can be, we should ref to things using
    # names.
    for child_collection in collection.children:
        if child_collection.hide_render:
            continue
        __handle_collection(context, cooked_scene, child_collection, xform)

    collection_cooker.cook_objects_into(context, xform, collection, cooked_scene)


def cook(context, scene):
    cooked_scene = json.loads(scene.worldspawn.values or '{}')

    # TODO: output path to the sky material, or emit the sky material as-is
    # cooked_scene['Sky'] = 'skies/industrial_sunset_puresky.ktx2'

    # def __handle_view_layer(layer_collection):
    #     if layer_collection.exclude:
    #         return

    #     for child in layer_collection.children:
    #         __handle_view_layer(child)

    #     __handle_collection(context, cooked_scene, layer_collection.collection, Matrix())

    # __handle_view_layer(scene.view_layers[0].layer_collection)

    tmp = __Cooker()
    tmp.cooked = {}
    tmp.entity = 1

    tmp.add_entity({'ScriptState': {'Worldspawn': cooked_scene}})

    __handle_collection(context, tmp, scene.collection, Matrix())

    cooked_scene = bpyutil.fixupdict(tmp.cooked) # pain
    with open(context.path_for_datablock(scene), 'wb') as f:
        json.dump(cooked_scene, util.UTF8Writer(f), indent='\t', default=bpyutil.asdasd)
