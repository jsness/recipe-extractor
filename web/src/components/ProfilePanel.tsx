import type { FormEvent } from "react";
import { Button, Card, Group, Select, Stack, Text, TextInput } from "@mantine/core";
import { Profile } from "../types";

type ProfilePanelProps = {
  profiles: Profile[];
  activeProfileId: string | null;
  isLoadingProfiles: boolean;
  isCreatingProfile: boolean;
  createProfileName: string;
  setCreateProfileName: (value: string) => void;
  onSelectProfile: (value: string | null) => void;
  onCreateProfile: (event: FormEvent<HTMLFormElement>) => void;
};

export const ProfilePanel = ({
  profiles,
  activeProfileId,
  isLoadingProfiles,
  isCreatingProfile,
  createProfileName,
  setCreateProfileName,
  onSelectProfile,
  onCreateProfile,
}: ProfilePanelProps) => (
  <Card withBorder radius="md" padding="lg">
    <Stack gap="md">
      <div>
        <Text fw={600}>Profile</Text>
        <Text c="dimmed" size="sm">
          Choose or create a profile before loading recipes or extracting new ones.
        </Text>
      </div>

      <Select
        data={profiles.map((profile) => ({ value: profile.id, label: profile.name }))}
        value={activeProfileId}
        onChange={onSelectProfile}
        placeholder={profiles.length === 0 ? "No profiles yet" : "Select a profile"}
        disabled={isLoadingProfiles || profiles.length === 0}
        searchable
        clearable={false}
      />

      <form onSubmit={onCreateProfile}>
        <Group align="flex-end">
          <TextInput
            value={createProfileName}
            onChange={(event) => setCreateProfileName(event.currentTarget.value)}
            placeholder="New profile name"
            disabled={isCreatingProfile || isLoadingProfiles}
            style={{ flex: 1 }}
          />
          <Button type="submit" loading={isCreatingProfile}>
            Add Profile
          </Button>
        </Group>
      </form>
    </Stack>
  </Card>
);
