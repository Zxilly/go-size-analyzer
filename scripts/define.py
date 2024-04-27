from enum import Flag, auto


class TestType(Flag):
    TEXT_TEST = auto()
    JSON_TEST = auto()
    HTML_TEST = auto()


class IntegrationTest:
    def __init__(self, name: str, path: str, type: TestType):
        self.name = name
        self.path = path
        self.type = type
