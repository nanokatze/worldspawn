import json


import bpyutil
import util


def deps(context, obj, dset):
    dset.add_product((context.path_for_datablock(obj), 'Object', obj.name))


def cook(context, obj):
    armature = obj.data

    # TODO: change this to soa
    inverse_bind_pose = {}

    for bone in armature.bones:
        inverse_bind_pose[bone.name] = bone.matrix_local.inverted_safe()

    inverse_bind_pose = bpyutil.fixupdict(inverse_bind_pose)

    with open(context.path_for_datablock(obj), 'wb') as f:
        json.dump(inverse_bind_pose, util.UTF8Writer(f), default=bpyutil.asdasd, indent='\t')
