import { Alert, Badge, Button, Card, CloseButton, Group, Loader, Stack, Text } from "@mantine/core";
import { ExtractionStatusResponse } from "../types";

type ExtractionCardProps = {
  extraction: ExtractionStatusResponse;
  isPolling: boolean;
  onDismiss: () => void;
  isTryingArchivedVersion?: boolean;
  onTryArchivedVersion?: () => void;
};

export const ExtractionCard = ({
  extraction,
  isPolling,
  onDismiss,
  isTryingArchivedVersion = false,
  onTryArchivedVersion,
}: ExtractionCardProps) => (
  <Card withBorder radius="md" padding="md">
    <Stack gap="xs">
      <Group justify="space-between" align="flex-start" wrap="nowrap">
        <Group justify="space-between" style={{ flex: 1, minWidth: 0 }}>
          <Text size="sm" c="dimmed">Extraction ID</Text>
          <Text size="sm" ff="monospace" style={{ overflowWrap: "anywhere" }}>{extraction.id}</Text>
        </Group>
        <CloseButton
          aria-label="Dismiss extraction"
          type="button"
          onClick={onDismiss}
          style={{ flexShrink: 0 }}
        />
      </Group>
      <Group justify="space-between">
        <Text size="sm" c="dimmed">Status</Text>
        <Badge color={extraction.status === "failed" ? "red" : extraction.status === "done" ? "green" : "cyan"}>
          {extraction.status}
        </Badge>
      </Group>
      {extraction.recipe_id && (
        <Group justify="space-between">
          <Text size="sm" c="dimmed">Recipe ID</Text>
          <Text size="sm" ff="monospace">{extraction.recipe_id}</Text>
        </Group>
      )}
      {extraction.error_message && (
        <Alert color="red" title="Failure">
          <Stack gap="xs">
            <Text size="sm">{extraction.error_message}</Text>
            {extraction.can_try_archived_source && onTryArchivedVersion && (
              <Group justify="space-between" align="center">
                <Text size="sm" c="dimmed">
                  This site may still be extractable from an archived copy.
                </Text>
                <Button
                  size="xs"
                  variant="light"
                  loading={isTryingArchivedVersion}
                  onClick={onTryArchivedVersion}
                >
                  Try archived version
                </Button>
              </Group>
            )}
          </Stack>
        </Alert>
      )}
      {isPolling && (
        <Group gap="xs">
          <Loader size="xs" />
          <Text size="sm" c="dimmed">Polling status...</Text>
        </Group>
      )}
    </Stack>
  </Card>
);
