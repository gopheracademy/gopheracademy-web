DOTS := $(wildcard *.dot)
SVGS := $(DOTS:.dot=.svg)
IMGDIR := ../../../static/postimages/advent-2018/interfaces-and-reflect/
.PHONY : clean

%.png : %.dot
	dot -Tpng $< >$@

%.svg : %.dot
	dot -Tsvg $< >$@

all: $(SVGS) $(PNGS)
	mkdir -p $(IMGDIR)
	cp $(SVGS) $(IMGDIR)

clean:
	rm -f $(SVGS)
