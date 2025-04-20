def generate():
    a = [1, 2, 3, 4, 5]
    for x in a:
        yield x


a = generate()
for x in a:
    print(x)
