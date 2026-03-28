import { Anchor, Button, Container, Group, Stack, Text, Title } from "@mantine/core";
import GithubLogo from "../icons/GithubLogo";

type LandingPageProps = {
  onLaunchApp: () => void;
};

export const LandingPage = ({ onLaunchApp }: LandingPageProps) => (
  <Container size="sm" pt="12vh">
    <Stack gap="xl">
      <div>
        <Title order={1}>Recipe Extractor</Title>
        <Text c="dimmed" mt="sm" size="lg">
          A self-hosted web app that pulls a recipe from any URL, strips away the ads and
          filler, and saves a clean structured copy to your own database.
        </Text>
      </div>

      <Group>
        <Button size="md" onClick={onLaunchApp}>
          Launch App
        </Button>
        <Anchor
          href="https://github.com/jsness/recipe-extractor"
          target="_blank"
          c="dimmed"
        >
          <Group gap="xs">
            <GithubLogo />
            <Text size="sm">GitHub</Text>
          </Group>
        </Anchor>
      </Group>
    </Stack>
  </Container>
);
