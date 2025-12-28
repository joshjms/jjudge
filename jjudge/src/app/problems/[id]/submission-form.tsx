"use client";

import { cpp } from "@codemirror/lang-cpp";
import { python } from "@codemirror/lang-python";
import { indentUnit } from "@codemirror/language";
import { vscodeLight } from "@uiw/codemirror-theme-vscode";
import CodeMirror from "@uiw/react-codemirror";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState, type FormEvent } from "react";

import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const languages = [
    { value: "cpp", label: "C++20" },
    { value: "python", label: "Python 3" },
];

type SubmissionFormProps = {
    problemId: number;
};

const getExtensions = (language: string) => {
    const indentationExtensions = [indentUnit.of("    ")];

    switch (language) {
        case "cpp":
            return [...indentationExtensions, cpp()];
        case "python":
            return [...indentationExtensions, python()];
        default:
            return [...indentationExtensions];
    }
};

export function SubmissionForm({ problemId }: SubmissionFormProps) {
    const auth = useAuth();
    const router = useRouter();
    const [language, setLanguage] = useState(languages[0].value);
    const [code, setCode] = useState<string>("");
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState(false);
    const extensions = useMemo(() => getExtensions(language), [language]);

    const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
        event.preventDefault();
        if (!code.trim()) {
            setError("Code is required.");
            return;
        }
        setIsSubmitting(true);
        setError(null);
        setSuccess(false);

        try {
            await api.post(
                "/submissions",
                {
                    problem_id: problemId,
                    language,
                    code,
                },
                {
                    headers: { Authorization: `Bearer ${auth.token}` },
                },
            );
            setSuccess(true);
            router.push(`/problems/${problemId}/submissions/mine`);
        } catch (err) {
            setError("Submission failed. Check your code and try again.");
        } finally {
            setIsSubmitting(false);
        }
    };

    if (!auth.token) {
        return (
            <div className="mt-12 bg-card/70 p-6 text-sm text-muted-foreground">
                Please <span className="text-primary underline"><Link href="/login">log in</Link></span> to submit a solution.
            </div>
        );
    }

    return (
        <form onSubmit={handleSubmit} className="mt-12">
            <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                    <h2 className="text-xl font-semibold">Submit your solution</h2>
                    <p className="text-sm text-muted-foreground">
                        Your code will be evaluated; pick a language and paste your solution below.
                    </p>
                </div>
                <label className="text-sm font-medium text-muted-foreground">
                    <span className="mr-2 text-foreground">Language</span>
                    <select
                        className="border border-border/60 bg-background px-3 py-1 text-sm"
                        value={language}
                        onChange={(e) => setLanguage(e.target.value)}
                    >
                        {languages.map((lang) => (
                            <option key={lang.value} value={lang.value}>
                                {lang.label}
                            </option>
                        ))}
                    </select>
                </label>
            </div>

            <div className="mt-5">
                <CodeMirror
                    value={code}
                    extensions={extensions}
                    onChange={(value) => setCode(value)}
                    theme={vscodeLight}
                    height="320px"
                    basicSetup={{
                        lineNumbers: true,
                        highlightActiveLine: true,
                        foldGutter: true,
                    }}
                    placeholder="Write your solution here..."
                    className="border border-border/70 bg-card/70"
                />
            </div>

            {error && (
                <p className="mt-4 border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                    {error}
                </p>
            )}
            {success && (
                <p className="mt-4 border border-emerald-500/50 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-700">
                    Submission created successfully.
                </p>
            )}

            <div className="mt-5 flex flex-wrap items-center gap-3">
                <Button type="submit" disabled={isSubmitting} className="rounded-none">
                    {isSubmitting ? "Submitting..." : "Submit solution"}
                </Button>
            </div>
        </form>
    );
}
