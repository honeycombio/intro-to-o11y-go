# Into to Observability: OpenTelemetry in Go

This application is here for you to try out tracing.
It consists of a microservice that calls itself, so you can simulate
a whole microservice ecosystem with just one service!

## What to do

Remix this app on Glitch.

(I recommend cloning the repo. Java in Glitch is verrrrry slooooow. This app on your computer is fast.)

In IntelliJ, add a run configuration for Maven target `spring-boot:run`. Then hit the home page at http://localhost:8080.

### 1. Autoinstrument!

This will make tracing happen in the Spring app with no code changes!
You'll see the web requests coming in. They'll even nest inside each other when the service calls itself. You will not
see anything specific to this app, like the query parameter on the request.

This magic happens through [instrumentation](https://docs.oracle.com/en/java/javase/11/docs/api/java.instrument/java/lang/instrument/Instrumentation.html) by a Java agent.
The agent gloms onto your Java app, recognizes Spring receiving HTTP requests, and emits events.

There's a general OpenTelemetry Java agent, and [Honeycomb wraps it into a version]
https://github.com/honeycombio/honeycomb-opentelemetry-java#agent-usage) that's easier to configure. we'll use that one.

#### Get the agent

Download the agent jar [from this direct link](https://github.com/honeycombio/honeycomb-opentelemetry-java/releases/download/v0.4.0/honeycomb-opentelemetry-javaagent-0.4.0-all.jar).
In Glitch, click Tools, the Command Line, and then do this:

`wget https://github.com/honeycombio/honeycomb-opentelemetry-java/releases/download/v0.4.0/honeycomb-opentelemetry-javaagent-0.4.0-all.jar`

`sync` (this tells glitch to notice what you did at the command line)

#### Attach the agent

The goal is to add a JVM argument: `-javaagent:honeycomb-opentelemetry-javaagent-0.4.0-all.jar`

To add this to `mvn spring-boot:run`,
open `pom.xml`, find the `plugin` block for `spring-boot-maven-plugin`, and 
add a `configuration` block like the one here:

```xml
 <plugin>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-maven-plugin</artifactId>
    <configuration>
        <agents>
            <agent>
                honeycomb-opentelemetry-javaagent-0.4.0-all.jar
            </agent>
        </agents>
    </configuration>
</plugin>
```

#### Configure the Agent

Finally, tell the agent how to send events to Honeycomb.
In `.env` in glitch or your run configuration in IntelliJ, add these
environment variables:

```
HONEYCOMB_API_KEY=replace-this-with-a-real-api-key
HONEYCOMB_DATASET=otel-java
SERVICE_NAME=fibonacci-microservice
SAMPLE_RATE=1
```

Get a Honeycomb API Key from your Team Settings in [Honeycomb](https://ui.honeycomb.io).
(find this by clicking on your profile in the lower-left corner.)

You can name the Honeycomb Dataset anything you want.

You can choose any Service Name you want.

The Sample Rate determines how many requests each saved trace represents; 1 means "keep all of them." Right now you want all of them.

#### See the results

Run the app. Activate the sequence of numbers.
Go to [Honeycomb](https://ui.honeycomb.io) and choose the Dataset you configured.

How many traces are there?

How many spans are in the traces?

Why are there so many??

Which trace has the most, and why is it different?

## 2. Customize a span

Let's make it easier to see what the "index" query parameter is.

To do this, change the code using the OpenTelemetry API.

### Bring in the OpenTelemetry API

Add these dependencies to add to `pom.xml`.

```xml
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-api</artifactId>
        <version>1.5.0</version>
    </dependency>
    <dependency>
        <groupId>io.opentelemetry</groupId>
        <artifactId>opentelemetry-extension-annotations</artifactId>
        <version>1.5.0</version>
    </dependency>
```

### Use the API in your code

Now in `FibonacciController.java`, in the `getFibonacciNumber` method, add the index parameter to the current Span:

```java
  Span span = Span.current();
  span.setAttribute("parameter.index", i);
```

Restart the app, make the sequence go, and find that field on the new spans.

Can you make the trace waterfall view show the index? What pattern does it show?

## 3. Create a custom span

Make the calculation into its own span, to see how much of the time spent on
this service is the meat: adding the fibonacci numbers.

Break out a method for creating the returned Fibonacci number, and add the
magical `@WithSpan` attribute.

Something like:

```java
  @WithSpan
  private FibonacciNumber calculate(int index, FibonacciNumber previous, FibonacciNumber oneBeforeThat) {
    return new FibonacciNumber(index, previous.fibonacciNumber + oneBeforeThat.fibonacciNumber);
  }
```

After a restart, do your traces show this extra span? Do you see the name of your method?
What percentage of the service time is spend in it?

