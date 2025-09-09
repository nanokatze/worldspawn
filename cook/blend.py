import bpy
import click

# TODO: give the vars proper names

# TODO: we'll want a context of sorts

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


class Context:

    # TODO: we need a notion of src_root/raw_root and dst_root/cooked_root.
    # Given that, we won't need output_directory


    def path_for_datablock(self, datablock):
        # TODO: clean this mess up

        library = datablock.library
        name = self.__name_for_datablock(datablock)
        return self.output_directory + ('../' + library.filepath[2:-6] + '/' if library else '') + name + '/' + bpy.path.clean_name(datablock.name)


    def __name_for_datablock(self, datablock):
        match datablock:
            case bpy.types.Collection():
                return 'collections'
            case bpy.types.Scene():
                return 'scenes'
            case bpy.types.Material():
                return 'materials'
            case bpy.types.Object():
                return 'geometries'
            case _:
                assert False, 'unsupported type {}'.format(datablock)


# TODO: move it to the end of the file probs
@click.command()
@click.option('-M', is_flag=True)
@click.option('-o')
@click.argument('blend')
@click.argument('datablock_type', required=False)
@click.argument('datablock_name', required=False)
def main(m, o, blend, datablock_type, datablock_name):
    bpy_context = bpy.context

    if m:
        dset = Dependencies()

        ctx = Context()
        ctx.bpy_context = bpy_context

        asd = blend

        import glob
        import pathlib
        import os
        # TODO: consume an explicit list instead of globbing
        for blend in glob.glob(asd + '/**/*.blend', recursive=True):
            dset.depends = {blend}

            ctx.output_directory = blend[len(asd):-len('.blend')] + '/'

            bpy.ops.wm.open_mainfile(filepath=blend)

            depsgraph = bpy_context.evaluated_depsgraph_get()

            from blender_cookers import collection as collection_cooker
            for datablock in bpy_context.blend_data.collections:
                if datablock.library:
                    continue
                settings = datablock.get('worldspawn', {})
                if not settings.get('export'):
                    continue
                collection_cooker.deps(ctx, datablock.evaluated_get(depsgraph), dset)
            from blender_cookers import material as material_cooker
            for datablock in bpy_context.blend_data.materials:
                settings = datablock.get('worldspawn', {})
                if settings.get('export'):
                    material_cooker.deps(ctx, datablock.evaluated_get(depsgraph), dset)
            import mesh_cooker
            for datablock in bpy_context.blend_data.objects:
                settings = datablock.get('worldspawn', {})
                if settings.get('export'):
                    mesh_cooker.deps(ctx, datablock.evaluated_get(depsgraph), dset)
            import scene_cooker
            for datablock in bpy_context.blend_data.scenes:
                settings = datablock.get('worldspawn', {})
                if not settings.get('export'):
                    continue
                scene_cooker.deps(ctx, datablock.evaluated_get(depsgraph), dset)

        with open('build.ninja', 'w') as f:
            f.write('rule blend\n')
            f.write('  command = python3.11 ../../cook/blend.py -o $o $in $datablock_type $datablock_name\n')
            for product, (datablock_type, datablock_name, depends) in dset.produces.items():
                f.write('\n')
                f.write(f'build {product}: blend {" ".join(depends)}\n')
                f.write(f'  raw = {asd}\n')
                f.write(f'  o = {os.path.split(os.path.split(product)[0])[0]}\n') # horrid
                f.write(f'  datablock_type = "{datablock_type}"\n')
                f.write(f'  datablock_name = "{datablock_name}"\n')
    else:
        bpy.ops.wm.open_mainfile(filepath=blend)

        blend_data = bpy_context.blend_data

        # TODO: outline this into a function
        datablocks = None
        cook = None
        match datablock_type:
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
                import mesh_cooker
                cook = mesh_cooker.cook

            case 'Scene':
                datablocks = blend_data.scenes
                import scene_cooker
                cook = scene_cooker.cook

        datablock = datablocks[datablock_name]

        depsgraph = bpy_context.evaluated_depsgraph_get()

        ctx = Context()
        ctx.bpy_context = bpy_context
        ctx.output_directory = o + '/'

        cook(ctx, datablock.evaluated_get(depsgraph))

main()
