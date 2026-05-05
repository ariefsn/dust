package cleaners

import (
	"github.com/ariefsn/dust/internal/cleaner"
)

// Gradle — wipe ~/.gradle/caches. No tool action; `./gradlew clean` is per-project.
func Gradle() cleaner.Cleaner {
	return pathBased{
		id:       "jvm/gradle",
		name:     "Gradle — caches",
		category: "JVM",
		resolvePath: func() string {
			return cleaner.Expand("~/.gradle/caches")
		},
	}
}

// Maven — wipe ~/.m2/repository. Maven re-downloads on next build.
func Maven() cleaner.Cleaner {
	return pathBased{
		id:       "jvm/maven",
		name:     "Maven — local repository",
		category: "JVM",
		resolvePath: func() string {
			return cleaner.Expand("~/.m2/repository")
		},
	}
}
