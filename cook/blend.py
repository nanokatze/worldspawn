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


    def path_for_datablock(self, datablock):
        name = self.__name_for_datablock(datablock)
        if name is None: # TODO: express ignored things somehow differently?
            return None
        return self.output_directory + name


    def __name_for_datablock(self, datablock):
        match datablock:
            case bpy.types.Scene():
                return 'Scene_' + bpy.path.clean_name(datablock.name)
            case bpy.types.Object():
                object = datablock
                match object.type:
                    case 'LIGHT':
                        return None
                    case 'MESH' | 'FONT':
                        return 'Geometry_' + bpy.path.clean_name(datablock.name)
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
        for blend in glob.glob(blend + '/**/*.blend', recursive=True):
            dset.depends = {blend}

            print(blend)
            ctx.output_directory = blend[len(asd):-len('.blend')] + '/'

            bpy.ops.wm.open_mainfile(filepath=blend)

            depsgraph = bpy_context.evaluated_depsgraph_get()

            import mesh_cooker
            for object in bpy_context.blend_data.objects:
                settings = object.get('worldspawn', {})
                if settings.get('export'):
                    mesh_cooker.deps(ctx, object.evaluated_get(depsgraph), dset)
            import scene_cooker
            for scene in bpy_context.blend_data.scenes:
                settings = scene.get('worldspawn', {})
                if settings.get('export'):
                    scene_cooker.deps(ctx, scene.evaluated_get(depsgraph), dset)

        with open('build.ninja', 'w') as f:
            f.write('rule blend\n')
            f.write('  command = python3.11 ../../cook/blend.py -o $o $in $datablock_type $datablock_name\n')
            for product, (datablock_type, datablock_name, depends) in dset.produces.items():
                f.write('\n')
                f.write(f'build {product}: blend {" ".join(depends)}\n')
                # TODO: should be output_directory
                f.write(f'  o = {os.path.split(product)[0]}\n') # horrid
                f.write(f'  datablock_type = "{datablock_type}"\n')
                f.write(f'  datablock_name = "{datablock_name}"\n')
    else:
        bpy.ops.wm.open_mainfile(filepath=blend)

        blend_data = bpy_context.blend_data

        # TODO: outline this into a function
        datablocks = None
        cook = None
        match datablock_type:
            case 'Material':
                datablocks = blend_data.materials
                import material_cooker
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
