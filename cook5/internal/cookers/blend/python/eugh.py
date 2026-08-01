import bpy
import os
import pathlib

import blender_schema


blender_schema.register_classes()


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

        name = self.__name_for_datablock(datablock)

        library = datablock.library
        if library:
            path = self.blend_filename[:-6] + '/../' + library.filepath[2:-6] + '/'
        else:
            path = self.blend_filename[:-6] + '/'
        path += name + '/' + bpy.path.clean_name(datablock.name)
        return path


    def __name_for_datablock(self, datablock):
        match datablock:
            case bpy.types.Action():
                return 'animations'
            case bpy.types.Collection():
                return 'prefabs' # TODO: rename to collections. Or merge with scenes.
            case bpy.types.Scene():
                return 'maps' # TODO: rename back to scenes
            case bpy.types.Material():
                return 'materials'
            case bpy.types.Object():
                match datablock.type:
                    case 'ARMATURE':
                        return 'skeletons'
                    case 'MESH' | 'FONT':
                        return 'geometries'
                    case _:
                        assert False, f'unsupported object type {datablock.type}'
            case _:
                assert False, f'unsupported type {datablock}'


def cook(out_dir, guh_dir, blend_filename):
    bpy.ops.wm.open_mainfile(filepath=guh_dir+'/'+blend_filename)

    blend_data = bpy.context.blend_data

    print('we shall cook into', out_dir)

    ctx = Context()
    ctx.bpy_context = bpy.context
    ctx.output_directory = out_dir + '/'
    ctx.project_directory = guh_dir + '/'
    ctx.blend_filename = blend_filename

    dset = Dependencies()

    depsgraph = bpy.context.evaluated_depsgraph_get()

    from blender_cookers import action_cooker
    for datablock in bpy.context.blend_data.actions:
        if datablock.library:
            continue
        if not datablock.worldspawn.export:
            continue
        action_cooker.deps(ctx, datablock.evaluated_get(depsgraph), dset)

    from blender_cookers import collection as collection_cooker
    for datablock in bpy.context.blend_data.collections:
        if datablock.library:
            continue
        if not datablock.worldspawn.export:
            continue
        collection_cooker.deps(ctx, datablock.evaluated_get(depsgraph), dset)

    from blender_cookers import material as material_cooker
    for datablock in bpy.context.blend_data.materials:
        if datablock.library:
            continue
        if not datablock.worldspawn.export:
            continue
        material_cooker.deps(ctx, datablock.evaluated_get(depsgraph), dset)

    from blender_cookers import armature as armature_cooker
    from blender_cookers import mesh as mesh_cooker
    for datablock in bpy.context.blend_data.objects:
        if datablock.library:
            continue
        if not datablock.worldspawn.export:
            continue
        if datablock.type == 'ARMATURE':
            armature_cooker.deps(ctx, datablock.evaluated_get(depsgraph), dset)
        else:
            mesh_cooker.deps(ctx, datablock.evaluated_get(depsgraph), dset)

    from blender_cookers import scene as scene_cooker
    for datablock in bpy.context.blend_data.scenes:
        if datablock.library:
            continue
        if not datablock.worldspawn.export:
            continue
        scene_cooker.deps(ctx, datablock.evaluated_get(depsgraph), dset)

    for product, (datablock_type, datablock_name, depends) in dset.produces.items():
        datablocks = None
        cook = None
        match datablock_type:
            case 'Action':
                datablocks = blend_data.actions
                from blender_cookers import action_cooker
                cook = action_cooker.cook

            case 'Collection':
                datablocks = blend_data.collections
                from blender_cookers import collection as collection_cooker
                cook = collection_cooker.cook

            case 'Material':
                datablocks = blend_data.materials
                from blender_cookers import material as material_cooker
                cook = material_cooker.cook

            case 'Object':
                datablocks = blend_data.objects

                datablock = datablocks[datablock_name]

                match datablock.type:
                    case 'ARMATURE':
                        from blender_cookers import armature as armature_cooker
                        cook = armature_cooker.cook

                    case 'MESH' | 'FONT':
                        from blender_cookers import mesh as mesh_cooker
                        cook = mesh_cooker.cook

            case 'Scene':
                datablocks = blend_data.scenes
                from blender_cookers import scene as scene_cooker
                cook = scene_cooker.cook

        datablock = datablocks[datablock_name]

        if not (isinstance(datablock, bpy.types.Object) and datablock.type == 'ARMATURE'):
            datablock = datablock.evaluated_get(depsgraph)

        cook(ctx, datablock)
