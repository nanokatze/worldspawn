import dataclasses


@dataclasses.dataclass
class Instruction:
    Bind: str
    Op: str
    Type: str
    Imm: str = None
    Args: list[str] = None


@dataclasses.dataclass
class Header:
    ParamTypes: list[str]
    Host: list[str]
    Program: list[Instruction]
