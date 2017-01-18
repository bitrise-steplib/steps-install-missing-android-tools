package analyzer

const testBuildGradleSDK21Tools2101FileContent = `
apply plugin: 'com.android.application'

android {
    compileSdkVersion 21
    buildToolsVersion "21.0.1"

    defaultConfig {
        applicationId "com.bitrise_io.sample_apps_android_simple"
        minSdkVersion 15
        targetSdkVersion 21
        versionCode 1
        versionName "1.0"
    }
    buildTypes {
        release {
            minifyEnabled false
            proguardFiles getDefaultProguardFile('proguard-android.txt'), 'proguard-rules.pro'
        }
    }
}

dependencies {
    compile fileTree(dir: 'libs', include: ['*.jar'])
    testCompile 'junit:junit:4.12'
    compile 'com.android.support:appcompat-v7:20.+'
}`

const testBuildGradleSDK24Tools2402SupportFileContent = `apply plugin: 'com.android.application'

android {
    compileSdkVersion 24
    buildToolsVersion "24.0.2"
    defaultConfig {
        applicationId "io.bitrise.androidbuildtoolstest"
        minSdkVersion 15
        targetSdkVersion 24
        versionCode 1
        versionName "1.0"
        testInstrumentationRunner "android.support.test.runner.AndroidJUnitRunner"
    }
    buildTypes {
        release {
            minifyEnabled false
            proguardFiles getDefaultProguardFile('proguard-android.txt'), 'proguard-rules.pro'
        }
    }
}

dependencies {
    compile fileTree(dir: 'libs', include: ['*.jar'])
    androidTestCompile('com.android.support.test.espresso:espresso-core:2.2.2', {
        exclude group: 'com.android.support', module: 'support-annotations'
    })
    compile 'com.android.support:appcompat-v7:24.2.1'
    testCompile 'junit:junit:4.12'
}`

const testBuildGradleSDK23Tools2303SupportPlayFileContent = `buildscript {
    repositories {
        maven { url 'https://maven.fabric.io/public' }
    }
    dependencies {
        classpath 'io.fabric.tools:gradle:1.+'
    }
}
apply plugin: 'com.android.application'
apply plugin: 'io.fabric'
repositories {
    maven { url 'https://maven.fabric.io/public' }
}
android {
    compileSdkVersion 23
    buildToolsVersion "23.0.3"
    defaultConfig {
        minSdkVersion 16
        targetSdkVersion 23
        applicationId "hu.kntcrw.cardsup"
        versionName "1.1.0"
        versionCode 15
        vectorDrawables.useSupportLibrary = true
    }
    signingConfigs {
        releaseConfig {
            keyAlias 'cardsup'
            keyPassword 'loyaltybox01'
            storeFile file("$System.env.HOME/keystores/cardsup_keystore.jks")
            storePassword 'loyaltybox01'
        }
    }
    buildTypes {
        release {
            minifyEnabled false
            proguardFiles getDefaultProguardFile('proguard-android.txt'), 'proguard-rules.pro'
            signingConfig signingConfigs.releaseConfig
        }
        debug {
            signingConfig signingConfigs.releaseConfig
        }
    }
}
ext {
    supportLibVersion = '23.4.0'
    awsAndroidSDKVersion = '2.2.8'
    googlePlayServiceVersion = '7.8.0'
}
dependencies {
    compile fileTree(dir: 'libs', include: ['*.jar'])
    testCompile 'junit:junit:4.12'
    compile project(':dynamicgrid')
    // Support Library
    compile "com.android.support:appcompat-v7:${supportLibVersion}"
    compile "com.android.support:design:${supportLibVersion}"
    compile "com.android.support:percent:${supportLibVersion}"
    compile "com.android.support:support-vector-drawable:${supportLibVersion}"
    compile "com.android.support:animated-vector-drawable:${supportLibVersion}"
    compile "com.android.support:support-annotations:${supportLibVersion}"
    // Google Play Services
    compile "com.google.android.gms:play-services-location:${googlePlayServiceVersion}"
    compile "com.google.android.gms:play-services-gcm:${googlePlayServiceVersion}"
    compile "com.google.android.gms:play-services-plus:${googlePlayServiceVersion}"
    // AWS SDK - https://docs.aws.amazon.com/mobile/sdkforandroid/developerguide/setup.html
    compile "com.amazonaws:aws-android-sdk-s3:${awsAndroidSDKVersion}"
    compile "com.amazonaws:aws-android-sdk-core:${awsAndroidSDKVersion}"
    compile "com.amazonaws:aws-android-sdk-cognito:${awsAndroidSDKVersion}"
    compile "com.amazonaws:aws-android-sdk-mobileanalytics:${awsAndroidSDKVersion}"
    compile "com.amazonaws:aws-android-sdk-sns:${awsAndroidSDKVersion}"
    compile "com.amazonaws:aws-android-sdk-ddb:${awsAndroidSDKVersion}"
    compile "com.amazonaws:aws-android-sdk-ddb-mapper:${awsAndroidSDKVersion}"
    // ZXing Android Embedded - https://github.com/journeyapps/zxing-android-embedded
    compile 'com.journeyapps:zxing-android-embedded:3.3.0@aar'
    compile 'com.google.zxing:core:3.2.1'
    // Facebook SDK for Android - https://github.com/facebook/facebook-android-sdk
    compile 'com.facebook.android:facebook-android-sdk:4.6.0'
    // Picasso - https://github.com/square/picasso
    compile 'com.squareup.picasso:picasso:2.5.2'
    // CircleImageView - https://github.com/hdodenhof/CircleImageView
    compile 'de.hdodenhof:circleimageview:2.0.0'
    // Crashlytics - https://fabric.io/kits/android/crashlytics/install
    compile('com.crashlytics.sdk.android:crashlytics:2.5.6@aar') {
        transitive = true;
    }
}`
