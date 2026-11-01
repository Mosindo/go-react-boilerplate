import type { MatchEvent, ReportReason } from "../api/types";

export type AuthStackParamList = {
  Auth: { mode?: "login" | "register" } | undefined;
  ForgotPassword: undefined;
};

export type RootStackParamList = {
  Tabs: undefined;
  Chat: { conversationId: string; userId: string; name: string };
  ProfileDetail: { userId: string; canSwipe?: boolean };
  EditProfile: undefined;
  Photos: undefined;
  Settings: undefined;
  Blocked: undefined;
  Notifications: undefined;
  Report: { userId: string; name: string; defaultReason?: ReportReason };
  DeleteAccount: undefined;
  MatchModal: MatchEvent;
};

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Me: undefined;
};

// Declaration requise par React Navigation pour typer useNavigation().
declare global {
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace ReactNavigation {
    // eslint-disable-next-line @typescript-eslint/no-empty-object-type
    interface RootParamList extends RootStackParamList {}
  }
}
