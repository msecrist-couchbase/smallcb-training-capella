import unittest
import time
import os
import sys
import urllib.parse
import yaml
import requests
from html import unescape


class CBLivePlaygroundRunCodeTest(unittest.TestCase):

    @classmethod
    def setUpClass(self):
        self.url = os.getenv("CBLIVE_URL","http://localhost:8080")
        self.code_dir = os.getenv("CODE_DIR", "../cmd/play-server/static/examples")
        self.cb_host = os.getenv("CB_HOST", "127.0.0.1")
        self.cb_user = os.getenv("CB_USER", "Administrator")
        self.cb_pwd = os.getenv("CB_PWD", "small-house-secret")

    def run_example_test(self, ex_id):
        code_path = "{}/{}.yaml".format(self.code_dir, ex_id)
        print('\nRunning {0} at {1} ...'.format((code_path.split("/")[-1]).split(".")[0], self.url))

        code_yaml = {}
        with open(code_path, 'r') as stream:
            try:
                code_yaml = yaml.safe_load(stream)
            except yaml.YAMLError as exc:
                print(exc)

        lang = code_yaml['lang']
        code = {}
        code['code'] = code_yaml['code'].replace('{{.Host}}', self.cb_host).replace('{{.CBUser}}',
                                            self.cb_user).replace('{{.CBPswd}}', self.cb_pwd)
        headers = {"Content-type": "application/x-www-form-urlencoded"}
        request_url = "{0}/run?lang={1}".format(self.url, lang)
        run_output = requests.post(request_url, data=code, headers = headers).text
        self.assertNotIn("Internal Server Error", run_output)
        output = unescape(run_output).split("<pre>")[1].split("</pre>")[0]
        print(output)
        return output

    # 1. Java examples
    def test_basic_java_kv_get(self):
        self.run_example_test(ex_id = "basic-java-kv-get")

    def test_ex_basic_java_query_rows(self):
        self.run_example_test(ex_id = "basic-java-query-rows")

    def test_basic_java_query_named_param(self):
        self.run_example_test(ex_id = "basic-java-query-named-param")

    def test_basic_java_query_positional_param(self):
        self.run_example_test(ex_id = "basic-java-query-positional-param")

    def test_basic_java_subdoc_lookup(self):
        self.run_example_test(ex_id = "basic-java-subdoc-lookup")

    def test_basic_java_subdoc_mutate(self):
        self.run_example_test(ex_id = "basic-java-subdoc-mutate")

    def test_basic_java_txn_kv_mutate(self):
        self.run_example_test(ex_id = "basic-java-txn-kv-mutate")

    def test_basic_java_txn_n1ql(self):
        self.run_example_test(ex_id = "basic-java-txn-n1ql")

    def test_basic_java_upsert(self):
        self.run_example_test(ex_id = "basic-java-upsert")

    # 2. Nodejs examples
    def test_basic_nodejs_kv_get(self):
        self.run_example_test(ex_id = "basic-nodejs-kv-get")

    def test_ex_basic_nodejs_query_rows(self):
        self.run_example_test(ex_id = "basic-nodejs-query-rows")

    def test_basic_nodejs_query_named_param(self):
        self.run_example_test(ex_id="basic-nodejs-query-named-param")

    def test_basic_nodejs_query_positional_param(self):
        self.run_example_test(ex_id = "basic-nodejs-query-positional-param")

    def test_basic_nodejs_subdoc_lookup(self):
        self.run_example_test(ex_id = "basic-nodejs-subdoc-lookup")

    def test_basic_nodejs_subdoc_mutate(self):
        self.run_example_test(ex_id = "basic-nodejs-subdoc-mutate")

    def test_basic_nodejs_upsert(self):
        self.run_example_test(ex_id = "basic-nodejs-upsert")

    # 3. Python examples
    def test_basic_py_kv_get(self):
        self.run_example_test(ex_id = "basic-py-kv-get")

    def test_ex_basic_py_query_rows(self):
        self.run_example_test(ex_id = "basic-py-query-rows")

    def test_basic_py_query_named_param(self):
        self.run_example_test(ex_id = "basic-py-query-named-param")

    def test_basic_py_query_positional_param(self):
        self.run_example_test(ex_id = "basic-py-query-positional-param")

    def test_basic_py_subdoc_lookup(self):
        self.run_example_test(ex_id = "basic-py-subdoc-lookup")

    def test_basic_py_subdoc_mutate(self):
        self.run_example_test(ex_id = "basic-py-subdoc-mutate")

    def test_basic_py_upsert(self):
        self.run_example_test(ex_id = "basic-py-upsert")

    # 4. Dotnet examples
    def test_basic_dotnet_kv_get(self):
        self.run_example_test(ex_id = "basic-dotnet-kv-get")

    def test_ex_basic_dotnet_query_rows(self):
        self.run_example_test(ex_id = "basic-dotnet-query-rows")

    def test_basic_dotnet_query_positional_param(self):
        self.run_example_test(ex_id = "basic-dotnet-query-positional-param")

    def test_basic_dotnet_subdoc_lookup(self):
        self.run_example_test(ex_id = "basic-dotnet-subdoc-lookup")

    def test_basic_dotnet_subdoc_mutate(self):
        self.run_example_test(ex_id = "basic-dotnet-subdoc-mutate")

    def test_basic_dotnet_upsert(self):
        self.run_example_test(ex_id = "basic-dotnet-upsert")

    # 5. PHP examples
    def test_basic_php_kv_get(self):
        self.run_example_test(ex_id = "basic-php-kv-get")

    def test_ex_basic_php_query_rows(self):
        self.run_example_test(ex_id = "basic-php-query-rows")

    def test_basic_php_query_positional_param(self):
        self.run_example_test(ex_id = "basic-php-query-positional-param")

    def test_basic_php_subdoc_lookup(self):
        self.run_example_test(ex_id = "basic-php-subdoc-lookup")

    def test_basic_php_subdoc_mutate(self):
        self.run_example_test(ex_id = "basic-php-subdoc-mutate")

    def test_basic_php_upsert(self):
        self.run_example_test(ex_id = "basic-php-upsert")

    # 6. Ruby examples
    def test_basic_ruby_kv_get(self):
        self.run_example_test(ex_id = "basic-ruby-kv-get")

    def test_ex_basic_ruby_query_rows(self):
        self.run_example_test(ex_id = "basic-ruby-query-rows")

    def test_basic_ruby_query_positional_param(self):
        self.run_example_test(ex_id = "basic-ruby-query-positional-param")

    def test_basic_ruby_subdoc_lookup(self):
        self.run_example_test(ex_id = "basic-ruby-subdoc-lookup")

    def test_basic_ruby_subdoc_mutate(self):
        self.run_example_test(ex_id = "basic-ruby-subdoc-mutate")

    def test_basic_ruby_upsert(self):
        self.run_example_test(ex_id = "basic-ruby-upsert")

    # 7. Scala examples
    def test_basic_scala_kv_get(self):
        self.run_example_test(ex_id = "basic-scala-kv-get")

    def test_ex_basic_scala_query_rows(self):
        self.run_example_test(ex_id = "basic-scala-query-rows")

    def test_basic_scala_query_positional_param(self):
        self.run_example_test(ex_id = "basic-scala-query-positional-param")

    def test_basic_scala_subdoc_lookup(self):
        self.run_example_test(ex_id = "basic-scala-subdoc-lookup")

    def test_basic_scala_subdoc_mutate(self):
        self.run_example_test(ex_id = "basic-scala-subdoc-mutate")

    def test_basic_scala_upsert(self):
        self.run_example_test(ex_id = "basic-scala-upsert")

    # 8. Go examples
    def test_basic_go_kv_get(self):
        self.run_example_test(ex_id = "basic-go-kv-get")

    def test_ex_basic_go_query_rows(self):
        self.run_example_test(ex_id = "basic-go-query-rows")

    def test_basic_go_query_positional_param(self):
        self.run_example_test(ex_id = "basic-go-query-positional-param")

    def test_basic_go_subdoc_lookup(self):
        self.run_example_test(ex_id = "basic-go-subdoc-lookup")

    def test_basic_go_subdoc_mutate(self):
        self.run_example_test(ex_id = "basic-go-subdoc-mutate")

    def test_basic_go_upsert(self):
        self.run_example_test(ex_id = "basic-go-upsert")

    # 9. C++ examples
    def test_basic_cc_kv_get(self):
        self.run_example_test(ex_id = "basic-cc-kv-get")

    def test_ex_basic_cc_query_rows(self):
        self.run_example_test(ex_id = "basic-cc-query-rows")

    def test_basic_cc_query_positional_param(self):
        self.run_example_test(ex_id = "basic-cc-query-positional-param")

    def test_basic_cc_subdoc_lookup(self):
        self.run_example_test(ex_id = "basic-cc-subdoc-lookup")

    def test_basic_cc_subdoc_mutate(self):
        self.run_example_test(ex_id = "basic-cc-subdoc-mutate")

    def test_basic_cc_upsert(self):
        self.run_example_test(ex_id = "basic-cc-upsert")

    @classmethod
    def tearDownClass(self):
        print("teardown...")

if __name__ == "__main__":
    unittest.main()


