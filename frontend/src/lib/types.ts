export type Status = "pending" | "running" | "succeeded" | "failed";

export type Template = {
  slug: string;
  name: string;
  description: string;
  source_repo: string;
  supports_lambda: boolean;
  required_checks: string[];
};

export type Repository = {
  id: string;
  name: string;
  description: string;
  template: string;
  private: boolean;
  with_lambda: boolean;
  status: Status;
  repo_url: string | null;
  error: string | null;
  requested_by_email: string;
  created_at: string;
  updated_at: string;
};

export type NewRepository = {
  name: string;
  description: string;
  template: string;
  private: boolean;
  with_lambda: boolean;
};
