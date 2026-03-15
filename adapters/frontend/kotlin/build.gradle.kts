plugins {
    kotlin("android") version "1.9.22"
    kotlin("plugin.serialization") version "1.9.22"
    id("com.android.library") version "8.2.2"
    id("maven-publish")
}

group = "com.rampart"
version = "0.1.0"

android {
    namespace = "com.rampart"
    compileSdk = 34

    defaultConfig {
        minSdk = 23
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
        consumerProguardFiles("consumer-rules.pro")
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }
}

dependencies {
    // Kotlin coroutines
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.7.3")

    // Kotlin serialization
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.6.3")

    // Ktor HTTP client
    implementation("io.ktor:ktor-client-core:2.3.8")
    implementation("io.ktor:ktor-client-okhttp:2.3.8")
    implementation("io.ktor:ktor-client-content-negotiation:2.3.8")
    implementation("io.ktor:ktor-serialization-kotlinx-json:2.3.8")

    // Encrypted storage
    implementation("androidx.security:security-crypto:1.1.0-alpha06")

    // Chrome Custom Tabs
    implementation("androidx.browser:browser:1.7.0")

    // AndroidX core
    implementation("androidx.core:core-ktx:1.12.0")

    // Testing
    testImplementation("junit:junit:4.13.2")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.7.3")
    testImplementation("io.ktor:ktor-client-mock:2.3.8")
}

publishing {
    publications {
        create<MavenPublication>("release") {
            groupId = "com.rampart"
            artifactId = "rampart-android"
            version = project.version.toString()

            pom {
                name.set("Rampart Android SDK")
                description.set("Native Android/Kotlin adapter for Rampart IAM — OAuth 2.0 PKCE, encrypted token storage, and Jetpack Compose support.")
                url.set("https://github.com/manimovassagh/rampart")

                licenses {
                    license {
                        name.set("MIT License")
                        url.set("https://opensource.org/licenses/MIT")
                    }
                }
            }
        }
    }
}
