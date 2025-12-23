import bpy
import struct
import json
import dataclasses

import util
import compiler
from cookers import material as material_cooker


def deps(context, datablock, dset):
    # TODO: should be handled in common code
    if datablock.library:
        return
    dset.add_product((context.path_for_datablock(datablock), 'Material', datablock.name))



def cook(ctx, material):
    assert material.use_nodes

    node_tree = material.node_tree

    material_output = None
    # aovs = []
    for n in node_tree.nodes:
        match n:
            case bpy.types.ShaderNodeOutputMaterial():
                material_output = n
    assert material_output is not None

    c = Context(node_tree)

    v_material_output = c.get((material_output, None))

    program = []
    for instr in c.builder.instructions:
        guh = material_cooker.Instruction(Bind=c.builder.names[instr.id],
                                          Type=instr.type,
                                          Op=instr.op,
                                          Imm=str(instr.imm) if instr.imm is not None else None,
                                          Args=[c.builder.names[a.id] for a in instr.args])
        program.append(dataclasses.asdict(guh))

    with open(ctx.path_for_datablock(material), 'wb') as f:
        json.dump({'Program': program}, util.UTF8Writer(f), indent='\t')


def _float32_bits(x):
    return struct.unpack('<I', struct.pack('<f', x))[0]


# TODO: rename this to something else pls
class Context:


    def __init__(self, node_tree):
        self.__links = _Links(node_tree)
        self.__node_values = dict()
        self.builder = compiler.Sea()


    # TODO: s should be the socket. I guess it could be the socket index?
    def get(self, x):
        v = self.__node_values.get(x) # TODO: should be hashed by the (node, output_name) pair
        if v is None:
            v = self.__translate(*x)
            self.__node_values[x] = v
        return v


    # This should be a user-provided lambda
    def __translate(self, node, output_name):
        inputs = _sockets_by_name(node.inputs)

        match node:
            case bpy.types.ShaderNodeOutputMaterial():
                assert output_name is None
                # TODO: return NullDF if surface is nil
                surface_df = self.__translate_output_socket(self.__links.to(inputs['Surface'])[0])
                # TODO: introduce an op that extracts BSDF and EDF from a surface
                surface_bsdf = surface_df
                surface_edf = self.builder.value('EDF', 'DFWeightedSum', None)
                surface = self.builder.value('Empty', 'MakeSurface', None, surface_bsdf, surface_edf)
                return surface

            case bpy.types.ShaderNodeBsdfPrincipled():
                assert output_name == 'BSDF'

                # TODO: properly translate from inputs['Normal']
                normal = self.builder.value('Array[3, Int[32]]', 'LoadShadingNormal', None)

                r = self.builder.value('Int[32]', 'IConst', _float32_bits(inputs['Base Color'].default_value[0]))
                g = self.builder.value('Int[32]', 'IConst', _float32_bits(inputs['Base Color'].default_value[1]))
                b = self.builder.value('Int[32]', 'IConst', _float32_bits(inputs['Base Color'].default_value[2]))
                base_color = self.builder.value('Array[3, Int[32]]', 'MakeArray', None, r, g, b)

                diffuse_bsdf = self.builder.value('BSDF', 'DiffuseBSDF', None, normal)
                diffuse_tinted = self.builder.value('BSDF', 'DFWeightedSum', None, base_color, diffuse_bsdf)

                return diffuse_tinted

            case _:
                assert False, 'unsupported node type {}'.format(type(node))


    def __translate_output_socket(self, socket):
        assert socket.is_output
        return self.get((socket.node, socket.name))


def _sockets_by_name(sockets):
    return {s.name: s for s in sockets}


# TODO: move to node_tree or w/e
class _Links:


    def __init__(self, node_tree):
        self.__to = {}

        for l in node_tree.links:
            # TODO: accumulate so that we support multiple
            self.__to[l.to_socket] = [l.from_socket]


    def to(self, socket):
        return self.__to.get(socket, [])

