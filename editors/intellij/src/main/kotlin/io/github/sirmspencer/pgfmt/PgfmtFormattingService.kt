package io.github.sirmspencer.pgfmt

import com.intellij.execution.ExecutionException
import com.intellij.execution.configurations.GeneralCommandLine
import com.intellij.execution.process.CapturingProcessHandler
import com.intellij.formatting.service.AsyncDocumentFormattingService
import com.intellij.formatting.service.AsyncFormattingRequest
import com.intellij.formatting.service.FormattingService
import com.intellij.psi.PsiFile
import java.nio.charset.StandardCharsets

private const val TIMEOUT_MS = 30_000

/**
 * Formats .sql files by piping the document through the pgfmt executable.
 *
 * The platform asks each registered formatting service in order and the
 * built-in one is registered last, so this service replaces the IDE's own
 * sql formatter for reformat code, actions on save, and commit reformatting.
 */
class PgfmtFormattingService : AsyncDocumentFormattingService() {

    // no FORMAT_FRAGMENTS: pgfmt formats whole statements, so reformatting a
    // selection falls back to the built-in formatter. no AD_HOC_FORMATTING:
    // typing and paste stay with the built-in formatter instead of starting a process.
    override fun getFeatures(): Set<FormattingService.Feature> = emptySet()

    override fun canFormat(file: PsiFile): Boolean =
        PgfmtSettings.getInstance().state.enabled &&
            file.virtualFile?.extension.equals("sql", ignoreCase = true)

    override fun getNotificationGroupId(): String = "pgfmt"

    override fun getName(): String = "pgfmt"

    override fun createFormattingTask(request: AsyncFormattingRequest): FormattingTask? {
        val executable = PgfmtExecutable.find()
        if (executable == null) {
            request.onError("pgfmt", "pgfmt was not found. Install it with Homebrew or set its path in Settings | Tools | pgfmt.")
            return null
        }
        val commandLine = GeneralCommandLine(executable).withCharset(StandardCharsets.UTF_8)
        request.ioFile?.parentFile?.let { commandLine.withWorkDirectory(it) }
        return PgfmtTask(commandLine, request)
    }

    private class PgfmtTask(
        private val commandLine: GeneralCommandLine,
        private val request: AsyncFormattingRequest,
    ) : FormattingTask {
        @Volatile private var handler: CapturingProcessHandler? = null
        @Volatile private var cancelled = false

        override fun run() {
            val process = try {
                CapturingProcessHandler(commandLine)
            } catch (error: ExecutionException) {
                request.onError("pgfmt", "could not start pgfmt: ${error.message}")
                return
            }
            handler = process
            // pgfmt reads all of stdin before writing anything, so writing first cannot deadlock
            process.processInput.use { it.write(request.documentText.toByteArray(StandardCharsets.UTF_8)) }
            val output = process.runProcess(TIMEOUT_MS)
            when {
                cancelled -> return
                output.isTimeout -> request.onError("pgfmt", "pgfmt did not finish within ${TIMEOUT_MS / 1000} seconds")
                output.exitCode == 0 -> request.onTextReady(output.stdout)
                else -> request.onError("pgfmt", output.stderr.trim().ifEmpty { "pgfmt exited with status ${output.exitCode}" })
            }
        }

        override fun cancel(): Boolean {
            cancelled = true
            handler?.destroyProcess()
            return true
        }

        override fun isRunUnderProgress(): Boolean = true
    }
}
