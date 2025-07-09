import os.path


class DirFS:

    def __init__(self, dir):
        self._dir = dir

    def _full_name(self, filename):
        return os.path.join(self._dir, _normpath('/' + filename).lstrip('/'))


# Normalize a forward slash-separated path by purely lexical processing.
def _normpath(path):
    sep = '/'

    root = path.startswith(sep)

    components = []
    for component in path.split(sep):
        match component:
            case '' | '.':
                pass
            case '..':
                if components:
                    components.pop()
            case _:
                components.append(component)

    if not components:
        components = ['.']

    new_path = sep.join(components)
    if root:
        new_path = sep + new_path
    return new_path
