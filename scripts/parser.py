from html.parser import HTMLParser


class DataParser(HTMLParser):
    def __init__(self):
        super().__init__()
        self.in_data = False
        self.data = None

    def handle_starttag(self, tag, attrs):
        if tag == "script":
            for attr in attrs:
                if attr[0] == "type" and attr[1] == "application/json":
                    self.in_data = True

    def handle_data(self, data):
        if self.in_data:
            self.data = data

    def handle_endtag(self, tag):
        if self.in_data:
            self.in_data = False

    def get_data(self):
        return self.data
