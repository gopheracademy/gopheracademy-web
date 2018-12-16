DOTS		:= $(wildcard *.dot)
SVGS		:= $(DOTS:.dot=.svg)
IMGDIR	:= ../../../static/postimages/advent-2018/interfaces-and-reflect/

.PHONY : clean

%.svg : %.dot
	dot -Tsvg -o$@ $<

all: images install

images: $(SVGS)
	mkdir -p $(IMGDIR)

install:
	cp $(SVGS) $(IMGDIR)

clean:
	rm -f $(SVGS)
