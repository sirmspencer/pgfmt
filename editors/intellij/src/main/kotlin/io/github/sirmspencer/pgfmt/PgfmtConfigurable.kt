package io.github.sirmspencer.pgfmt

import com.intellij.openapi.options.BoundConfigurable
import com.intellij.openapi.ui.DialogPanel
import com.intellij.ui.dsl.builder.COLUMNS_LARGE
import com.intellij.ui.dsl.builder.bindSelected
import com.intellij.ui.dsl.builder.bindText
import com.intellij.ui.dsl.builder.columns
import com.intellij.ui.dsl.builder.panel

/** The Settings | Tools | pgfmt page. */
class PgfmtConfigurable : BoundConfigurable("pgfmt") {

    private val options = PgfmtSettings.getInstance().state

    override fun createPanel(): DialogPanel = panel {
        row {
            checkBox("Format .sql files with pgfmt")
                .bindSelected(options::enabled)
        }
        row("pgfmt executable:") {
            textField()
                .bindText({ options.path.orEmpty() }, { options.path = it })
                .columns(COLUMNS_LARGE)
                .comment("Leave empty to use pgfmt from the PATH or the Homebrew bin directory.")
        }
    }
}
