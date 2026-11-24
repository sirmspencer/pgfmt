plugins {
    id("org.jetbrains.kotlin.jvm")
    id("org.jetbrains.intellij.platform")
}

group = "io.github.sirmspencer"
version = providers.gradleProperty("pluginVersion").get()

kotlin {
    jvmToolchain(21)
}

dependencies {
    intellijPlatform {
        intellijIdeaCommunity(providers.gradleProperty("platformVersion"))
    }
}

intellijPlatform {
    pluginConfiguration {
        version = providers.gradleProperty("pluginVersion")
        ideaVersion {
            sinceBuild = providers.gradleProperty("pluginSinceBuild")
            untilBuild = provider { null }
        }
    }
    pluginVerification {
        ides {
            // the installed ides, so verification does not download others
            local(file("/Applications/IntelliJ IDEA CE.app"))
            local(file("/Applications/DataGrip.app"))
        }
    }
}

intellijPlatformTesting {
    runIde {
        // ./gradlew runDataGrip checks that pgfmt replaces the built-in sql formatter
        register("runDataGrip") {
            localPath = file("/Applications/DataGrip.app")
        }
    }
}
