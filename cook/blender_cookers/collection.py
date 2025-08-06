import util
import json

from mathutils import Matrix

def deps(context, datablock, dset):
    for child in datablock.children:
        if datablock.hide_render:
            continue
        deps(ctx, child, dset)

    import mesh_cooker

    for obj in datablock.objects:
        if obj.hide_render:
            continue
        mesh_cooker.deps(context, obj, dset)

    dset.add_product((context.path_for_datablock(datablock), 'Collection', datablock.name))


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
def cook_objects_into(context, xform, collection, cooked_scene, hack):
    for obj in collection.objects:
        if obj.hide_render:
            continue

        comps = dict(obj.get('worldspawn', {}).get('components', {}))

        T, R, S = (xform @ obj.matrix_world).decompose()
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

        if obj.instance_collection is not None:
            # TODO: do not inline the collection instances!!!
            #hack(context, cooked_scene, obj.instance_collection, obj.matrix_world)
            comps['CollectionInstance'] = {
                'Filename': context.path_for_datablock(obj.instance_collection),
            }

        # comps['Name'] = obj.name

        cooked_scene.add_entity(comps)


def cook(context, datablock):
    tmp = __Cooker()
    tmp.cooked = {}
    tmp.entity = 1

    __handle_collection(context, tmp, datablock, Matrix())

    cooked = util.fixupdict(tmp.cooked) # pain
    with open(context.path_for_datablock(datablock), 'wb') as f:
        json.dump(cooked, util.UTF8Writer(f), indent='\t', default=util.asdasd)



# TODO: in the future we'll need to go over objects two times: once to assign
# IDs and once more to actually collect
def __handle_collection(context, cooked_scene, collection, xform):
    for child_collection in collection.children:
        if child_collection.hide_render:
            continue
        __handle_collection(context, cooked_scene, child_collection, xform)

    cook_objects_into(context, xform, collection, cooked_scene, __handle_collection)
