package net.messeb.OpenTelemetry.controller;

import io.micrometer.observation.Observation;
import io.micrometer.observation.ObservationRegistry;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;
import java.util.NoSuchElementException;
import java.util.concurrent.ThreadLocalRandom;

/**
 * Demo endpoints — each one shows a different tracing scenario in Kibana APM.
 *
 * Every HTTP request is automatically traced by Spring Boot's Micrometer
 * integration. No manual span creation is needed for basic HTTP tracing.
 *
 * The ObservationRegistry is used to add custom child spans inside a request
 * (see /api/orders) so you can see span hierarchies in the APM trace view.
 */
@RestController
@RequestMapping("/api")
public class MyController {

    private static final Logger log = LoggerFactory.getLogger(MyController.class);

    private final ObservationRegistry registry;

    public MyController(ObservationRegistry registry) {
        this.registry = registry;
    }

    /**
     * Fast endpoint — shows a clean, successful trace with minimal latency.
     * Good baseline for comparing against the slow and error endpoints.
     */
    @GetMapping("/hello")
    public Map<String, Object> hello() {
        log.info("hello called");
        return Map.of(
                "message", "Hello from Spring Boot!",
                "service", "spring-otel-demo"
        );
    }

    /**
     * Slow endpoint — artificially delays the response.
     * In Kibana APM, this shows as a high-latency transaction.
     * Use ?ms=2000 to see it appear in the "slow transactions" view.
     */
    @GetMapping("/slow")
    public Map<String, Object> slow(@RequestParam(defaultValue = "800") long ms)
            throws InterruptedException {
        log.info("slow called — sleeping {}ms", ms);
        Thread.sleep(ms);
        return Map.of("message", "slow response", "delayMs", ms);
    }

    /**
     * Error endpoint — throws a RuntimeException.
     * In Kibana APM, this appears as a failed transaction and creates
     * an entry in Observability → APM → your service → Errors.
     */
    @GetMapping("/error")
    public Map<String, String> error() {
        throw new RuntimeException("Simulated error — check APM Errors tab for the stack trace");
    }

    /**
     * Nested spans endpoint — wraps each "order fetch" in its own child span.
     * In Kibana APM, open this trace to see the waterfall view with child spans:
     *
     *   GET /api/orders
     *     └── fetch-orders
     *           ├── fetch-order-1  (80ms)
     *           ├── fetch-order-2  (120ms)
     *           └── fetch-order-3  (60ms)
     */
    @GetMapping("/orders")
    public Map<String, Object> orders() {
        return Observation.createNotStarted("fetch-orders", registry).observe(() -> {
            List<Map<String, Object>> items = List.of(
                    fetchOrder(1, 80),
                    fetchOrder(2, 120),
                    fetchOrder(3, 60)
            );
            log.info("fetched {} orders", items.size());
            return Map.of("orders", items, "total", items.size());
        });
    }

    /**
     * Parameterized endpoint — shows how path parameters appear in APM.
     * The route is grouped as GET /api/orders/{id} in the APM transactions list,
     * not as separate entries per ID.
     *
     * Try /api/orders/101 or higher to trigger a 404.
     */
    @GetMapping("/orders/{id}")
    public Map<String, Object> order(@PathVariable int id) {
        log.info("fetching order id={}", id);
        if (id > 100) {
            throw new NoSuchElementException("Order " + id + " not found");
        }
        return Map.of(
                "id", id,
                "status", "shipped",
                "items", ThreadLocalRandom.current().nextInt(1, 6)
        );
    }

    // ── helpers ─────────────────────────────────────────────────────────────────

    private Map<String, Object> fetchOrder(int id, long delayMs) {
        return Observation.createNotStarted("fetch-order-" + id, registry).observe(() -> {
            try {
                Thread.sleep(delayMs);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
            return Map.of("id", id, "status", "ok", "delayMs", delayMs);
        });
    }
}
