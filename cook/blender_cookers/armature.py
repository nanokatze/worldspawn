import json


import bpyutil
import util


def deps(context, obj, dset):
    dset.add_product((context.path_for_datablock(obj), 'Object', obj.name))


def cook(context, obj):
    armature = obj.data

    # TODO: change everything to plain arrays
    parent = {}
    bind_pose = {}

    for bone in armature.bones:
        if bone.parent is not None:
            parent[bone.name] = bone.parent.name
        bind_pose[bone.name] = bone.matrix_local

    skeleton = {
        'Parent': parent,
        'BindPose': bind_pose,
    }

    skeleton = bpyutil.fixupdict(skeleton)

    with open(context.path_for_datablock(obj), 'wb') as f:
        json.dump(skeleton, util.UTF8Writer(f), default=bpyutil.asdasd, indent='\t')
