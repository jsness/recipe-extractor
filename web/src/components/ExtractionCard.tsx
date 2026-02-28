import { Alert, Badge, Card, Group, Loader, Stack, Text } from "@mantine/core";
import { ExtractionStatusResponse } from "../types";

type ExtractionCardProps = {
  extraction: ExtractionStatusResponse;
  isPolling: boolean;
};

export const ExtractionCard = ({ extraction, isPolling }: ExtractionCardProps) => (
  <Card withBorder radius="md" padding="md">
    <Stack gap="xs">
      <Group justify="space-between">
        <Text size="sm" c="dimmed">Extraction ID</Text>
        <Text size="sm" ff="monospace">{extraction.id}</Text>
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
          {extraction.error_message}
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
