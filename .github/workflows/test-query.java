import com.couchbase.client.core.error.DocumentNotFoundException;
import com.couchbase.client.java.*;
import com.couchbase.client.java.kv.*;

class Program {
  public static void main(String[] args) {
    var cluster = Cluster.connect(
      "couchbase://{{.Host}}", "{{.CBUser}}", "{{.CBPswd}}"
    );

    var bucket = cluster.bucket("travel-sample");
    var collection = bucket.defaultCollection();

    try {
      var result = collection.get("airline_10");
      System.out.println(result.toString());

    } catch (DocumentNotFoundException ex) {
      System.out.println("Document not found!");
    }
  }
}

