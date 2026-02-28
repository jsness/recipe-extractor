import React from "react";
import { Button, Group, TextInput } from "@mantine/core";

type ExtractFormProps = {
  url: string;
  setURL: (v: string) => void;
  onSubmit: (e: React.FormEvent<HTMLFormElement>) => void;
  isSubmitting: boolean;
};

export const ExtractForm = ({ url, setURL, onSubmit, isSubmitting }: ExtractFormProps) => (
  <form onSubmit={onSubmit}>
    <Group align="flex-end">
      <TextInput
        type="url"
        placeholder="https://example.com/recipe"
        value={url}
        onChange={(event) => setURL(event.target.value)}
        disabled={isSubmitting}
        style={{ flex: 1 }}
        required
      />
      <Button type="submit" loading={isSubmitting}>
        Extract
      </Button>
    </Group>
  </form>
);
