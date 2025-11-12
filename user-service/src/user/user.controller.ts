import {
  Controller,
  Get,
  Post,
  Body,
  Patch,
  Param,
  Delete,
  Query,
  ParseIntPipe,
  HttpCode,
  HttpStatus,
} from "@nestjs/common";
import { UserService } from "./user.service";
import { CreateUserDto } from "./dto/create-user.dto";
import { UpdateUserDto } from "./dto/update-user.dto";
import { UpdatePreferencesDto } from "./dto";

@Controller("users")
export class UserController {
  constructor(private readonly userService: UserService) {}

  @Post()
  async create(@Body() createUserDto: CreateUserDto) {
    const user = await this.userService.create(createUserDto);
    return {
      message: "User created successfully",
      data: user,
    };
  }

  @Get()
  async findAll(
    @Query("page", new ParseIntPipe({ optional: true })) page: number = 1,
    @Query("limit", new ParseIntPipe({ optional: true })) limit: number = 10,
  ) {
    const result = await this.userService.findAll(page, limit);
    return {
      message: "Users retrieved successfully",
      ...result,
    };
  }

  @Get(":id")
  async findOne(@Param("id") id: string) {
    const user = await this.userService.findOne(id);
    return {
      message: "User retrieved successfully",
      data: user,
    };
  }

  @Patch(":id")
  async update(@Param("id") id: string, @Body() updateUserDto: UpdateUserDto) {
    const user = await this.userService.update(id, updateUserDto);
    return {
      message: "User updated successfully",
      data: user,
    };
  }

  @Patch(":id/preferences")
  async updatePreferences(
    @Param("id") id: string,
    @Body() updatePreferencesDto: UpdatePreferencesDto,
  ) {
    const user = await this.userService.updatePreferences(
      id,
      updatePreferencesDto,
    );
    return {
      message: "Preferences updated successfully",
      data: user,
    };
  }

  @Delete(":id")
  @HttpCode(HttpStatus.NO_CONTENT)
  async remove(@Param("id") id: string) {
    await this.userService.remove(id);
  }
}
