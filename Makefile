unit-test: go-code-tester
	GITHUB_OUTPUT=/dev/null \
	./go-code-tester 85 "." "" "true" "" "" ""

clean:
	rm -f go-code-tester *.log *.out cover*

go-code-tester:
	git clone --depth 1 git@github.com:dell/actions.git temp-repo
	cp temp-repo/go-code-tester/entrypoint.sh ./go-code-tester
	chmod +x go-code-tester
	rm -rf temp-repo
