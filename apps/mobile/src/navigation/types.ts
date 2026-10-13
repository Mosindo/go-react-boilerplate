import type { NavigatorScreenParams } from "@react-navigation/native";

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Notifications: undefined;
  Profile: undefined;
};

export type RootStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Conversation: { conversationId: string; userId: string; name: string; photoId?: string | null };
  ProfileDetail: { userId: string };
  EditProfile: undefined;
  Preferences: undefined;
  Photos: undefined;
  Settings: undefined;
};

declare global {
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace ReactNavigation {
    // eslint-disable-next-line @typescript-eslint/no-empty-object-type
    interface RootParamList extends RootStackParamList {}
  }
}
