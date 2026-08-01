@{
    # scripts/*.ps1 are standalone, interactive installers that a user runs (or
    # pipes into iex) once. They are not a PowerShell module, so the rules below
    # describe a shape these files are not trying to have. Everything else stays
    # on, including PSReviewUnusedParameter and PSAvoidAssignmentToAutomaticVariable.
    ExcludeRules = @(
        # The installer's whole job is to print a coloured report to the person
        # watching it. Write-Output would pollute the pipeline and Write-Information
        # is invisible by default.
        'PSAvoidUsingWriteHost',

        # Print-Step, Download-Binary, Normalize-WindowsPathEntry and friends are
        # file-local helpers, never exported and never invoked as cmdlets, so the
        # approved-verb vocabulary does not apply.
        'PSUseApprovedVerbs',

        # Same reason: these helpers are not cmdlets, and the script itself is the
        # confirmation surface, having been run deliberately by the user.
        'PSUseShouldProcessForStateChangingFunctions',

        # Get-GovmanProfilePaths returns several paths; the plural is accurate.
        'PSUseSingularNouns',

        # install.ps1 is fetched and piped straight into iex; a BOM on the wire is
        # more likely to break that than to help.
        'PSUseBOMForUnicodeEncodedFile',

        # False positive against this layout: -Quiet, -Version and -Help are script
        # level parameters read inside functions (install.ps1:53, :146, :531 and
        # uninstall.ps1:503). PowerShell resolves those through dynamic scoping,
        # which the rule's static analysis does not model, so it reports every one
        # of them as unused.
        'PSReviewUnusedParameter'
    )
}
