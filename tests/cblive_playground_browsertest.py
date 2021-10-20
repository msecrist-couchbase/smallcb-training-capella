import unittest
from selenium import webdriver
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.common.by import By
import time
import os


class CBLivePlaygroundTest(unittest.TestCase):
    @classmethod
    def setUpClass(self):
        driver_options = os.getenv("DRIVER_OPTIONS", "ui")
        if driver_options != "ui":
            print("Setting chrome driver options={}".format(driver_options))
            options = webdriver.ChromeOptions()
            executable_path = ""
            for driver_option in driver_options.split(","):
                if "executable_path" in driver_option:
                    executable_path = driver_option.split("=")[1]
                print(driver_option)
                options.add_argument(driver_option)
            if executable_path:
                self.browser = webdriver.Chrome(
                    executable_path=executable_path, options=options
                )
            else:
                self.browser = webdriver.Chrome(options=options)
        else:
            self.browser = webdriver.Chrome()
        self.TIMEOUT = 10
        self.url = os.getenv("CBLIVE_URL", "https://couchbase.live")

    def navigate_to_home(self):
        self.browser.implicitly_wait(self.TIMEOUT)
        self.browser.get(self.url)

    def test_title(self):
        print("Running title test...")
        browser = self.browser
        self.navigate_to_home()
        self.assertIn("Couchbase Playground", browser.title)
        assert "Error" not in browser.page_source

    def run_example_test(self, page, ex_id):
        print("\nRunning {} ...".format(ex_id))

        browser = self.browser
        self.navigate_to_home()

        try:
            # click on the language selection
            # sometimes need to click twice to make it work
            browser.find_element_by_css_selector(f"a[href='{page}']").click()
            browser.find_element_by_css_selector(f"a[href='{page}']").click()
        except:
            pass

        try:
            # Check for the example in the language examples
            WebDriverWait(browser, self.TIMEOUT).until(
                EC.presence_of_element_located((By.CSS_SELECTOR, f"a[href$='{ex_id}']"))
            )
        except:
            pass

        # Load the example in the language examples
        browser.find_element_by_css_selector(f"a[href$='{ex_id}']").click()
        # browser.find_element_by_css_selector("#" + ex_id).click()
        WebDriverWait(browser, self.TIMEOUT).until(
            EC.presence_of_element_located((By.CSS_SELECTOR, "input#run.run"))
        )

        # browser.find_element_by_css_selector('input#run.run').click()
        # browser.execute_script("arguments[0].click();", input)
        # webdriver.ActionChains(browser).move_to_element(input).click(input).perform()
        browser.find_element_by_xpath('//*[@id="run" and @class="run"]').click()
        time.sleep(3)
        WebDriverWait(browser, self.TIMEOUT).until(
            EC.visibility_of_element_located((By.ID, "code-output"))
        )
        browser.switch_to.frame("output")
        WebDriverWait(browser, self.TIMEOUT).until(
            EC.presence_of_element_located((By.XPATH, "//html/body/pre"))
        )
        output = browser.find_element_by_xpath("//html/body/pre")
        run_output = output.text
        self.assertNotIn("Internal Server Error", run_output)
        self.assertNotIn("Couchbase Error:", run_output)
        print("output=" + run_output)
        return run_output

    # 1. Java examples
    def test_basic_java_kv_get(self):
        self.run_example_test(
            page="/examples/basic-java-kv-get", ex_id="basic-java-kv-get"
        )

    def test_ex_basic_java_query_rows(self):
        self.run_example_test(
            page="/examples/basic-java-kv-get", ex_id="basic-java-query-rows"
        )

    def test_basic_java_query_named_param(self):
        self.run_example_test(
            page="/examples/basic-java-kv-get", ex_id="basic-java-query-named-param"
        )

    def test_basic_java_query_positional_param(self):
        self.run_example_test(
            page="/examples/basic-java-kv-get",
            ex_id="basic-java-query-positional-param",
        )

    def test_basic_java_subdoc_lookup(self):
        self.run_example_test(
            page="/examples/basic-java-kv-get", ex_id="basic-java-subdoc-lookup"
        )

    def test_basic_java_subdoc_mutate(self):
        self.run_example_test(
            page="/examples/basic-java-kv-get", ex_id="basic-java-subdoc-mutate"
        )

    def test_basic_java_txn_kv_mutate(self):
        self.run_example_test(
            page="/examples/basic-java-kv-get", ex_id="basic-java-txn-kv-mutate"
        )

    def test_basic_java_txn_n1ql(self):
        self.run_example_test(
            page="/examples/basic-java-kv-get", ex_id="basic-java-txn-n1ql"
        )

    def test_basic_java_upsert(self):
        self.run_example_test(
            page="/examples/basic-java-kv-get", ex_id="basic-java-upsert"
        )

    # 2. Nodejs examples
    def test_basic_nodejs_kv_get(self):
        self.run_example_test(
            page="/examples/basic-nodejs-kv-get", ex_id="basic-nodejs-kv-get"
        )

    def test_ex_basic_nodejs_query_rows(self):
        self.run_example_test(
            page="/examples/basic-nodejs-kv-get", ex_id="basic-nodejs-query-rows"
        )

    def test_basic_nodejs_query_named_param(self):
        self.run_example_test(
            page="/examples/basic-nodejs-kv-get",
            ex_id="basic-nodejs-query-named-param",
        )

    def test_basic_nodejs_query_positional_param(self):
        self.run_example_test(
            page="/examples/basic-nodejs-kv-get",
            ex_id="basic-nodejs-query-positional-param",
        )

    def test_basic_nodejs_subdoc_lookup(self):
        self.run_example_test(
            page="/examples/basic-nodejs-kv-get", ex_id="basic-nodejs-subdoc-lookup"
        )

    def test_basic_nodejs_subdoc_mutate(self):
        self.run_example_test(
            page="/examples/basic-nodejs-kv-get", ex_id="basic-nodejs-subdoc-mutate"
        )

    def test_basic_nodejs_upsert(self):
        self.run_example_test(
            page="/examples/basic-nodejs-kv-get", ex_id="basic-nodejs-upsert"
        )

    # 3. Python examples
    def test_basic_py_kv_get(self):
        self.run_example_test(page="/examples/basic-py-kv-get", ex_id="basic-py-kv-get")

    def test_ex_basic_py_query_rows(self):
        self.run_example_test(
            page="/examples/basic-py-kv-get", ex_id="basic-py-query-rows"
        )

    def test_basic_py_query_named_param(self):
        self.run_example_test(
            page="/examples/basic-py-kv-get", ex_id="basic-py-query-named-param"
        )

    def test_basic_py_query_positional_param(self):
        self.run_example_test(
            page="/examples/basic-py-kv-get", ex_id="basic-py-query-positional-param"
        )

    def test_basic_py_subdoc_lookup(self):
        self.run_example_test(
            page="/examples/basic-py-kv-get", ex_id="basic-py-subdoc-lookup"
        )

    def test_basic_py_subdoc_mutate(self):
        self.run_example_test(
            page="/examples/basic-py-kv-get", ex_id="basic-py-subdoc-mutate"
        )

    def test_basic_py_upsert(self):
        self.run_example_test(page="/examples/basic-py-kv-get", ex_id="basic-py-upsert")

    # 4. Dotnet examples
    def test_basic_dotnet_kv_get(self):
        self.run_example_test(
            page="/examples/basic-dotnet-kv-get", ex_id="basic-dotnet-kv-get"
        )

    def test_ex_basic_dotnet_query_rows(self):
        self.run_example_test(
            page="/examples/basic-dotnet-kv-get", ex_id="basic-dotnet-query-rows"
        )

    def test_basic_dotnet_query_named_param(self):
        self.run_example_test(
            page="/examples/basic-dotnet-kv-get", ex_id="basic-dotnet-query-named-param"
        )

    def test_basic_dotnet_query_positional_param(self):
        self.run_example_test(
            page="/examples/basic-dotnet-kv-get",
            ex_id="basic-dotnet-query-positional-param",
        )

    def test_basic_dotnet_subdoc_lookup(self):
        self.run_example_test(
            page="/examples/basic-dotnet-kv-get", ex_id="basic-dotnet-subdoc-lookup"
        )

    def test_basic_dotnet_subdoc_mutate(self):
        self.run_example_test(
            page="/examples/basic-dotnet-kv-get", ex_id="basic-dotnet-subdoc-mutate"
        )

    def test_basic_dotnet_upsert(self):
        self.run_example_test(
            page="/examples/basic-dotnet-kv-get", ex_id="basic-dotnet-upsert"
        )

    # 5. PHP examples
    def test_basic_php_kv_get(self):
        self.run_example_test(
            page="/examples/basic-php-kv-get", ex_id="basic-php-kv-get"
        )

    def test_ex_basic_php_query_rows(self):
        self.run_example_test(
            page="/examples/basic-php-kv-get", ex_id="basic-php-query-rows"
        )

    def test_basic_php_query_named_param(self):
        self.run_example_test(
            page="/examples/basic-php-kv-get", ex_id="basic-php-query-named-param"
        )

    def test_basic_php_query_positional_param(self):
        self.run_example_test(
            page="/examples/basic-php-kv-get", ex_id="basic-php-query-positional-param"
        )

    def test_basic_php_subdoc_lookup(self):
        self.run_example_test(
            page="/examples/basic-php-kv-get", ex_id="basic-php-subdoc-lookup"
        )

    def test_basic_php_subdoc_mutate(self):
        self.run_example_test(
            page="/examples/basic-php-kv-get", ex_id="basic-php-subdoc-mutate"
        )

    def test_basic_php_upsert(self):
        self.run_example_test(
            page="/examples/basic-php-kv-get", ex_id="basic-php-upsert"
        )

    # # 6. Ruby examples
    # def test_basic_ruby_kv_get(self):
    #     self.run_example_test(page="/examples/basic-rb-kv-get", ex_id="basic-rb-kv-get")

    # def test_ex_basic_ruby_query_rows(self):
    #     self.run_example_test(
    #         page="/examples/basic-rb-kv-get", ex_id="basic-rb-query-rows"
    #     )

    # def test_basic_ruby_query_named_param(self):
    #     self.run_example_test(
    #         page="/examples/basic-rb-kv-get", ex_id="basic-rb-query-named-param"
    #     )

    # def test_basic_ruby_query_positional_param(self):
    #     self.run_example_test(
    #         page="/examples/basic-rb-kv-get",
    #         ex_id="basic-rb-query-positional-param",
    #     )

    # def test_basic_ruby_subdoc_lookup(self):
    #     self.run_example_test(
    #         page="/examples/basic-rb-kv-get", ex_id="basic-rb-subdoc-lookup"
    #     )

    # def test_basic_ruby_subdoc_mutate(self):
    #     self.run_example_test(
    #         page="/examples/basic-rb-kv-get", ex_id="basic-rb-subdoc-mutate"
    #     )

    # def test_basic_ruby_upsert(self):
    # self.run_example_test(page="/examples/basic-rb-kv-get", ex_id="basic-rb-upsert")

    # 7. Scala examples
    def test_basic_scala_kv_get(self):
        self.run_example_test(
            page="/examples/basic-scala-kv-get", ex_id="basic-scala-kv-get"
        )

    def test_ex_basic_scala_query_rows(self):
        self.run_example_test(
            page="/examples/basic-scala-kv-get", ex_id="basic-scala-query-rows"
        )

    def test_basic_scala_query_named_param(self):
        self.run_example_test(
            page="/examples/basic-scala-kv-get", ex_id="basic-scala-query-named-param"
        )

    def test_basic_scala_query_positional_param(self):
        self.run_example_test(
            page="/examples/basic-scala-kv-get",
            ex_id="basic-scala-query-positional-param",
        )

    def test_basic_scala_subdoc_lookup(self):
        self.run_example_test(
            page="/examples/basic-scala-kv-get", ex_id="basic-scala-subdoc-lookup"
        )

    def test_basic_scala_subdoc_mutate(self):
        self.run_example_test(
            page="/examples/basic-scala-kv-get", ex_id="basic-scala-subdoc-mutate"
        )

    def test_basic_scala_upsert(self):
        self.run_example_test(
            page="/examples/basic-scala-kv-get", ex_id="basic-scala-upsert"
        )

    # 8. Go examples
    def test_basic_go_kv_get(self):
        self.run_example_test(page="/examples/basic-go-kv-get", ex_id="basic-go-kv-get")

    def test_ex_basic_go_query_rows(self):
        self.run_example_test(
            page="/examples/basic-go-kv-get", ex_id="basic-go-query-rows"
        )

    def test_basic_go_query_named_param(self):
        self.run_example_test(
            page="/examples/basic-go-kv-get", ex_id="basic-go-query-named-param"
        )

    def test_basic_go_query_positional_param(self):
        self.run_example_test(
            page="/examples/basic-go-kv-get", ex_id="basic-go-query-positional-param"
        )

    def test_basic_go_subdoc_lookup(self):
        self.run_example_test(
            page="/examples/basic-go-kv-get", ex_id="basic-go-subdoc-lookup"
        )

    def test_basic_go_subdoc_mutate(self):
        self.run_example_test(
            page="/examples/basic-go-kv-get", ex_id="basic-go-subdoc-mutate"
        )

    def test_basic_go_upsert(self):
        self.run_example_test(page="/examples/basic-go-kv-get", ex_id="basic-go-upsert")

    # 9. C++ examples
    def test_basic_cc_kv_get(self):
        self.run_example_test(page="/examples/basic-cc-kv-get", ex_id="basic-cc-kv-get")

    def test_ex_basic_cc_query_rows(self):
        self.run_example_test(
            page="/examples/basic-cc-kv-get", ex_id="basic-cc-query-rows"
        )

    def test_basic_cc_query_named_param(self):
        self.run_example_test(
            page="/examples/basic-cc-kv-get", ex_id="basic-cc-query-named-param"
        )

    def test_basic_cc_query_positional_param(self):
        self.run_example_test(
            page="/examples/basic-cc-kv-get", ex_id="basic-cc-query-positional-param"
        )

    def test_basic_cc_subdoc_lookup(self):
        self.run_example_test(
            page="/examples/basic-cc-kv-get", ex_id="basic-cc-subdoc-lookup"
        )

    def test_basic_cc_subdoc_mutate(self):
        self.run_example_test(
            page="/examples/basic-cc-kv-get", ex_id="basic-cc-subdoc-mutate"
        )

    def test_basic_cc_upsert(self):
        self.run_example_test(page="/examples/basic-cc-kv-get", ex_id="basic-cc-upsert")

    @classmethod
    def tearDownClass(self):
        self.browser.close()


if __name__ == "__main__":
    unittest.main()
