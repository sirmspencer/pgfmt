package io.github.sirmspencer.pgfmt

import com.intellij.lang.CustomAutoFormatSyntaxErrorsVerifier
import com.intellij.psi.PsiElement

/**
 * Lets reformat on save run on .sql files that the IDE's parser marks as
 * invalid. Without it, the platform skips automatic formatting of any file
 * with a syntax error, so a missing semicolon that pgfmt would repair stops
 * the file from being formatted at all. pgfmt reports the errors it cannot
 * repair itself.
 */
class PgfmtSyntaxErrorsVerifier : CustomAutoFormatSyntaxErrorsVerifier {
    override fun isApplicable(context: PsiElement): Boolean = formatsWithPgfmt(context.containingFile)

    override fun checkValid(context: PsiElement): Boolean = true
}
