package io.github.sirmspencer.pgfmt

import com.intellij.execution.configurations.PathEnvironmentVariableUtil
import com.intellij.openapi.components.BaseState
import com.intellij.openapi.components.Service
import com.intellij.openapi.components.SimplePersistentStateComponent
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage
import com.intellij.openapi.components.service
import java.io.File

/** Application-wide settings, stored in pgfmt.xml in the IDE config directory. */
@Service(Service.Level.APP)
@State(name = "PgfmtSettings", storages = [Storage("pgfmt.xml")])
class PgfmtSettings : SimplePersistentStateComponent<PgfmtSettings.Options>(Options()) {

    class Options : BaseState() {
        var enabled by property(true)

        // empty means look up pgfmt on the path and in the homebrew bin directories
        var path by string("")
    }

    companion object {
        fun getInstance(): PgfmtSettings = service()
    }
}

object PgfmtExecutable {
    // apps started from the dock may not see the shell path, so check homebrew directly
    private val homebrewPaths = listOf("/opt/homebrew/bin/pgfmt", "/usr/local/bin/pgfmt")

    fun find(): String? {
        val configured = PgfmtSettings.getInstance().state.path?.trim().orEmpty()
        if (configured.isNotEmpty()) {
            return configured
        }
        PathEnvironmentVariableUtil.findInPath("pgfmt")?.let { return it.absolutePath }
        return homebrewPaths.firstOrNull { File(it).canExecute() }
    }
}
