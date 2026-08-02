import bpy
import os
import pathlib

import blender_schema


blender_schema.register_classes()


from blender_cookers import action_cooker
from blender_cookers import armature as armature_cooker
from blender_cookers import collection as collection_cooker
from blender_cookers import material as material_cooker
from blender_cookers import mesh as mesh_cooker
from blender_cookers import scene as scene_cooker
def _get_cooker(datablock):
    match datablock:
        case bpy.types.Action():
            return action_cooker
        case bpy.types.Collection():
            return collection_cooker
        case bpy.types.Material():
            return material_cooker
        case bpy.types.Object():
            match datablock.type:
                case 'ARMATURE':
                    return armature_cooker
                case 'MESH' | 'FONT':
                    return mesh_cooker
        case bpy.types.Scene():
            return scene_cooker
    return None # TODO: raise an exception instead that we don't know how to cook this


class Dependencies:

    def __init__(self):
        self.produces = dict()
        self.depends = set()


    def add_product(self, what):
        if what[0] is None:
            return
        self.produces[what[0]] = (what[1], what[2], self.depends)


    def add_dependency(self, d):
        self.depends.add(d)


# TODO: make this an equivalent of Artifacts in the Go part
class Context:

    # TODO: we need a notion of src_root/raw_root and dst_root/cooked_root.
    # Given that, we won't need output_directory


    def create(self, filename):
        fullpath = self.output_directory + '/' + filename
        pathlib.Path(os.path.dirname(fullpath)).mkdir(parents=True, exist_ok=True)
        return open(fullpath, 'wb')


    def path_for_datablock(self, datablock):
        # TODO: clean this mess up

        library = datablock.library
        if library:
            path = self.blend_filename[:-6] + '/../' + library.filepath[2:-6] + '/'
        else:
            path = self.blend_filename[:-6] + '/'
        path += _get_cooker(datablock).DIR + '/' + bpy.path.clean_name(datablock.name)
        return path


def cook(out_dir, guh_dir, blend_filename):
    bpy.ops.wm.open_mainfile(filepath=guh_dir+'/'+blend_filename)

    blend_data = bpy.context.blend_data

    ctx = Context()
    ctx.bpy_context = bpy.context
    ctx.output_directory = out_dir + '/'
    ctx.project_directory = guh_dir + '/'
    ctx.blend_filename = blend_filename

    dset = Dependencies()

    depsgraph = bpy.context.evaluated_depsgraph_get()

    for datablocks in [
        bpy.context.blend_data.actions,
        bpy.context.blend_data.collections,
        bpy.context.blend_data.materials,
        bpy.context.blend_data.objects,
        bpy.context.blend_data.scenes,
    ]:
        for datablock in datablocks:
            if datablock.library:
                continue
            if not datablock.worldspawn.export:
                continue
            cooker = _get_cooker(datablock)
            if cooker is None:
                continue
            cooker.deps(ctx, datablock.evaluated_get(depsgraph), dset)

    for product, (datablock_type, datablock_name, depends) in dset.produces.items():
        datablocks = None
        match datablock_type:
            case 'Action':
                datablocks = blend_data.actions
            case 'Collection':
                datablocks = blend_data.collections
            case 'Material':
                datablocks = blend_data.materials
            case 'Object':
                datablocks = blend_data.objects
            case 'Scene':
                datablocks = blend_data.scenes

        datablock = datablocks[datablock_name]

        if not (isinstance(datablock, bpy.types.Object) and datablock.type == 'ARMATURE'):
            datablock = datablock.evaluated_get(depsgraph)

        _get_cooker(datablock).cook(ctx, datablock)
