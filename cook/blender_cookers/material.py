import bpy
import struct
import json
import dataclasses

import util
import compiler
import bpyutil
from cookers import material as material_cooker


def deps(context, datablock, dset):
    # TODO: should be handled in common code
    if datablock.library:
        return
    dset.add_product((context.path_for_datablock(datablock), 'Material', datablock.name))



def cook(ctx, material):
    assert material.use_nodes

    node_tree = material.node_tree

    material_output_node = None
    # aovs = []
    for n in node_tree.nodes:
        match n:
            case bpy.types.ShaderNodeOutputMaterial():
                material_output_node = n
    assert material_output_node is not None

    sea = compiler.Sea()
    c = Compiler(compiler.Builder(sea), node_tree)

    material_output = c.compile((material_output_node, None))

    program = []
    for instr in sea.instructions:
        guh = material_cooker.Instruction(Bind=sea.names[instr.id],
                                          Op=instr.op,
                                          Type=instr.type,
                                          Imm=str(instr.imm) if instr.imm is not None else None,
                                          Args=[sea.names[a.id] for a in instr.args])
        program.append(dataclasses.asdict(guh))

    with open(ctx.path_for_datablock(material), 'wb') as f:
        json.dump({'Params': c.params, 'Preamble': c.preamble, 'Program': program}, util.UTF8Writer(f), indent='\t')


# TODO: split the material-specific parts out into a separate class and move
# stuff that can be reused for geonodes etc into node_tree.py?
class Compiler:


    # TODO: should this get the entire builder? I guess yeah
    def __init__(self, builder, node_tree):
        self.__links = bpyutil.Links(node_tree)
        self.params = [] # TODO: delet
        self.preamble = [] # TODO: delet
        self.__builder = builder
        self.__compiled_nodes = dict()


    # TODO: remove this in favor of just dumping everything into a single
    # representation. We should have the compiler figure out preamble for us
    # later.
    def __add_param(self, type, attribute):
        idx = len(self.params)
        self.params.append(type)
        # TODO: preamble will at one point become an actual program that will run on host
        self.preamble.append(attribute)
        return idx


    def compile(self, x):
        v = self.__compiled_nodes.get(x)
        if v is None:
            v = self.__compiled_nodes[x] = self.__compile_impl(*x)
        return v


    def __compile_impl(self, node, output_name):
        node_inputs = bpyutil.sockets_by_name(node.inputs)

        match node:
            case bpy.types.ShaderNodeAttribute():
                param_idx = self.__add_param('AttributeDescriptor', node.attribute_name)
                descriptor = self.__builder.value('LoadParameter', 'AttributeDescriptor', param_idx)
                value = self.__builder.value('LoadAttribute', 'Array[4, Int[32]]', None, descriptor)

                match output_name:
                    case 'Color':
                        return value

                    case _:
                        assert False, 'not implemented ({})'.format(output_name)

            case bpy.types.ShaderNodeCombineColor():
                red = self.__compile_input_socket(node_inputs['Red'], self.__socket_default_value)
                green = self.__compile_input_socket(node_inputs['Green'], self.__socket_default_value)
                blue = self.__compile_input_socket(node_inputs['Blue'], self.__socket_default_value)
                alpha = self.__builder.value('IConst', 'Int[32]', util.float32_bits(1))
                return self.__builder.value('MakeArray', 'Array[4, Int[32]]', None, red, green, blue, alpha)

            case bpy.types.ShaderNodeBsdfPrincipled():
                assert output_name == 'BSDF'

                in_base_color = node_inputs['Base Color']
                in_normal = node_inputs['Normal']

                # TODO: properly translate from node_inputs['Normal']
                normal = self.__builder.value('LoadShadingNormal', 'Array[3, Int[32]]', None)

                base_color = self.__compile_input_socket(in_base_color, self.__socket_default_value)

                diffuse_bsdf = self.__builder.value('DiffuseBSDF', 'BSDF', None, normal)
                diffuse_tinted = self.__builder.value('DFWeightedSum', 'BSDF', None, base_color, diffuse_bsdf)

                return diffuse_tinted

            case bpy.types.ShaderNodeOutputMaterial():
                assert output_name is None
                in_surface = node_inputs['Surface']
                # TODO: return NullDF if surface is nil
                surface_df = self.__compile_output_socket(self.__links.to(in_surface)[0])
                # TODO: introduce an op that extracts BSDF and EDF from a surface
                surface_bsdf = surface_df
                surface_edf = self.__builder.value('DFWeightedSum', 'EDF', None)
                surface = self.__builder.value('MakeSurface', 'Nothing', None, surface_bsdf, surface_edf)
                return surface

            case _:
                assert False, 'unsupported node type {}'.format(type(node))

        assert False, 'unreachable'


    def __compile_input_socket(self, socket, default):
        assert not socket.is_output
        links = self.__links.to(socket)
        match len(links):
            case 0:
                return default(socket)
            case 1:
                return self.__compile_output_socket(links[0])
            case _:
                assert False, 'socket has multiple inputs ({})'.format(len(links))


    def __compile_output_socket(self, socket):
        assert socket.is_output
        return self.compile((socket.node, socket.name))


    def __socket_default_value(self, socket):
        match socket:
            case bpy.types.NodeSocketColor():
                r = self.__builder.value('IConst', 'Int[32]', util.float32_bits(socket.default_value[0]))
                g = self.__builder.value('IConst', 'Int[32]', util.float32_bits(socket.default_value[1]))
                b = self.__builder.value('IConst', 'Int[32]', util.float32_bits(socket.default_value[2]))
                a = self.__builder.value('IConst', 'Int[32]', util.float32_bits(socket.default_value[3]))
                return self.__builder.value('MakeArray', 'Array[4, Int[32]]', None, r, g, b, a)

            case bpy.types.NodeSocketFloatFactor():
                return self.__builder.value('IConst', 'Int[32]', util.float32_bits(socket.default_value))

            case _:
                assert False, 'unsupported socket type {}'.format(type(socket))

