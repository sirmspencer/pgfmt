package io.github.sirmspencer.pgfmt

import com.intellij.openapi.application.PathManager
import com.intellij.openapi.util.SystemInfo
import com.intellij.util.EnvironmentUtil
import com.intellij.util.system.CpuArch
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

    /** The configured path, then the bundled binary, then pgfmt on the path or in homebrew. */
    fun find(): String? {
        val configured = PgfmtSettings.getInstance().state.path?.trim().orEmpty()
        if (configured.isNotEmpty()) {
            return configured
        }
        bundled()?.let { return it }
        onPath()?.let { return it }
        return homebrewPaths.firstOrNull { File(it).canExecute() }
    }

    private val executableName = if (SystemInfo.isWindows) "pgfmt.exe" else "pgfmt"

    /** pgfmt on the path; on macOS this is the login shell's path, which apps from the dock lack. */
    private fun onPath(): String? {
        val path = EnvironmentUtil.getValue("PATH") ?: System.getenv("PATH") ?: return null
        return path.split(File.pathSeparator)
            .filter { it.isNotBlank() }
            .map { File(it, executableName) }
            .firstOrNull { it.isFile && it.canExecute() }
            ?.absolutePath
    }

    /** The binary shipped in the plugin's bin/<os>-<arch> folder for this machine. */
    private fun bundled(): String? {
        // the plugin's classes are in <plugin>/lib/<jar>, and the binaries in <plugin>/bin
        val jar = PathManager.getJarPathForClass(PgfmtExecutable::class.java) ?: return null
        val pluginDir = File(jar).parentFile?.parentFile ?: return null
        val os = when {
            SystemInfo.isMac -> "darwin"
            SystemInfo.isLinux -> "linux"
            SystemInfo.isWindows -> "windows"
            else -> return null
        }
        val arch = if (CpuArch.isArm64()) "arm64" else "amd64"
        val file = File(pluginDir, "bin/$os-$arch/$executableName")
        if (!file.isFile) {
            return null
        }
        // unpacking a plugin zip can drop the executable bit
        if (!file.canExecute()) {
            file.setExecutable(true)
        }
        return file.absolutePath
    }
}
