import { IsBoolean, IsOptional } from "class-validator";

export class UpdatePreferencesDto {
  @IsBoolean()
  @IsOptional()
  email_enabled?: boolean;

  @IsBoolean()
  @IsOptional()
  push_enabled?: boolean;
}
