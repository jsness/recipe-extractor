import { useEffect, useRef, useState } from "react";
import { Alert, Button, Group, Paper, Stack, Text, Textarea, Title } from "@mantine/core";

type RecipeReminderProps = {
  reminder?: string;
  isEditing: boolean;
  onEditingChange: (editing: boolean) => void;
  onSave: (reminder: string) => Promise<boolean>;
  onFocusAdd: () => void;
};

export const RecipeReminder = ({
  reminder, isEditing, onEditingChange, onSave, onFocusAdd,
}: RecipeReminderProps) => {
  const [draft, setDraft] = useState(reminder ?? "");
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState("");
  const editorRef = useRef<HTMLTextAreaElement>(null);
  const editRef = useRef<HTMLButtonElement>(null);
  const mounted = useRef(true);
  const focusAfterSave = useRef(false);

  useEffect(() => {
    mounted.current = true;
    return () => { mounted.current = false; };
  }, []);

  useEffect(() => {
    if (isEditing) {
      setDraft(reminder ?? "");
      setError("");
      editorRef.current?.scrollIntoView({ block: "center", behavior: "smooth" });
      editorRef.current?.focus({ preventScroll: true });
    }
  }, [isEditing]);

  useEffect(() => {
    if (!isEditing && focusAfterSave.current) {
      focusAfterSave.current = false;
      if (reminder) editRef.current?.focus();
      else onFocusAdd();
    }
  }, [isEditing, reminder, onFocusAdd]);

  const save = async (value: string) => {
    setIsSaving(true);
    setError("");
    try {
      const applied = await onSave(value);
      if (mounted.current && applied) {
        focusAfterSave.current = true;
        onEditingChange(false);
      }
    } catch (failure) {
      if (mounted.current) {
        setError(failure instanceof Error ? failure.message : "Unable to save reminder.");
      }
    } finally {
      if (mounted.current) setIsSaving(false);
    }
  };

  if (!reminder && !isEditing) return null;

  return (
    <Paper withBorder p="md" bg="transparent">
      <Stack gap="sm">
        <Title order={4}>Reminder</Title>
        {isEditing ? (
          <>
            <Textarea
              ref={editorRef}
              label="Reminder for next time"
              value={draft}
              onChange={(event) => setDraft(event.currentTarget.value)}
              autosize
              minRows={3}
              disabled={isSaving}
            />
            <Group justify="flex-end">
              <Button
                onClick={() => void save(draft)}
                loading={isSaving}
                disabled={draft.trim() === (reminder ?? "")}
              >
                Save
              </Button>
              <Button
                variant="subtle"
                disabled={isSaving}
                onClick={() => {
                  setError("");
                  focusAfterSave.current = true;
                  onEditingChange(false);
                }}
              >
                Cancel
              </Button>
            </Group>
          </>
        ) : (
          <Text size="sm" style={{ whiteSpace: "pre-wrap", overflowWrap: "anywhere" }}>{reminder}</Text>
        )}
        {error && <Alert color="red" title="Reminder could not be saved" role="alert">{error}</Alert>}
        {reminder && (
          <Group justify="flex-end" gap="xs" wrap="nowrap">
            {!isEditing && (
              <Button
                ref={editRef}
                variant="light"
                size="xs"
                disabled={isSaving}
                onClick={() => {
                  setDraft(reminder ?? "");
                  setError("");
                  onEditingChange(true);
                }}
              >
                Edit
              </Button>
            )}
            <Button
              color="red"
              variant="subtle"
              size="xs"
              disabled={isSaving}
              onClick={() => {
                if (window.confirm("Remove this reminder? This cannot be undone.")) {
                  void save("");
                }
              }}
            >
              Remove
            </Button>
          </Group>
        )}
      </Stack>
    </Paper>
  );
};
