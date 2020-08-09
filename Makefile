build:
	docker build -t smallcb .

run:
	docker run -p 8091-8094:8091-8094 -p 11210:11210 smallcb
